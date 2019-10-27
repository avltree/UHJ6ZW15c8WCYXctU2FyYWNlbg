package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

func main() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer, middleware.URLFormat)
	r.Use(middleware.Timeout(60))
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(ConstrainPayload)

	r.Route("/api/fetcher", func(r chi.Router) {
		r.Get("/", getObjectList)
		r.Post("/", postObject)
		r.Route("/{objectId}", func(r chi.Router) {
			r.Use(getObjectContext)
			r.Delete("/", deleteObject)
			r.Get("/history", getObjectHistory)
		})
	})

	go work()
	http.ListenAndServe(":8080", r)
}

func ConstrainPayload(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"actual": r.ContentLength,
			"max": 1 << (10 * 2),
		}).Debug("Content length")

		if r.ContentLength > 1 << (10 * 2) {
			log.Error("Payload exceeding 1 MB posted")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			w.Write([]byte(""))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getObjectList(w http.ResponseWriter, r *http.Request)  {
	var list []render.Renderer

	for _, o := range getObjects() {
		list = append(list, ObjectResponse{Object: o})
	}

	if 0 == len(list) {
		w.Write([]byte("[]"))
		return
	}

	// TODO handle highlighted errors
	render.RenderList(w, r, list)
}

func getObjectContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stringId := chi.URLParam(r, "objectId")
		log.WithFields(log.Fields{"id": stringId}).Info("Object id retrieved from URL")

		if "" == stringId {
			// TODO use a function for rendering errors
			log.Error("No object id provided")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(""))
			return
		}

		id, err := strconv.Atoi(stringId)

		if nil != err || id <= 0 {
			// TODO use a function for rendering errors
			log.WithFields(log.Fields{"id": stringId}).Error("Provided id is not a positive integer")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(""))
			return
		}

		o := Object{Id: int64(id)}
		err = o.findAndFill()

		if nil != err {
			// TODO use a function for rendering errors
			log.WithFields(log.Fields{
				"id": id,
				"err": err,
			}).Error("Error searching object in the database")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(""))
			return
		}

		ctx := context.WithValue(r.Context(), "object", o)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func postObject(w http.ResponseWriter, r *http.Request) {
	data := &ObjectRequest{}

	if e := render.Bind(r, data); e != nil {
		log.WithFields(log.Fields{"error": e}).Error("Error creating new Object")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(""))
		return
	}

	o := data.Object

	if err := o.save(); err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("Error saving object")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(""))
		return
	}

	render.Status(r, http.StatusCreated)
	render.Render(w, r, ObjectIdResponse{
		Object: o,
	})
}

func deleteObject(w http.ResponseWriter, r *http.Request) {
	o := r.Context().Value("object").(Object)
	log.WithFields(log.Fields{"object": o}).Info("Deleting object")

	if err := o.delete(); nil != err {
		log.WithFields(log.Fields{"error": err.Error()}).Error("Error deleting object")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(""))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(""))
	return
}

func getObjectHistory(w http.ResponseWriter, r *http.Request)  {
	o := r.Context().Value("object").(Object)
	log.WithFields(log.Fields{"object": o}).Info("Retrieving object history")

	var list []render.Renderer
	// TODO handle error
	history, _ := o.getHistory()

	for _, oh := range history {
		list = append(list, ObjectHistoryResponse{oh})
	}

	if 0 == len(list) {
		w.Write([]byte("[]"))
		return
	}

	// TODO handle highlighted errors
	render.RenderList(w, r, list)
}

// TODO delete those interface or offer some abstraction
type Saveable interface {
	save() error
}

type Findable interface {
	findAndFill() error
}

type Deletable interface {
	delete() error
}

// TODO separate this into another package and the response classes elsewhere
type Object struct {
	Id int64 `json:"id"`
	Url string `json:"url"`
	Interval int `json:"interval"`
	LastCheck time.Time `json:"last_check"`
}

func (o *Object) delete() error {
	db := getDbConnection()
	defer db.Close()

	_, err := db.Query("delete from objects where id = ?", o.Id)

	return err
}

func (o *Object) findAndFill() error {
	db := getDbConnection()
	defer db.Close()

	// TODO handle errors
	result, err := db.Query("select url, `interval` from objects where id = ?", o.Id)
	// TODO this and the find all should use the same "hydration"
	result.Next()
	err = result.Scan(&o.Url, &o.Interval)

	return err
}

func (o *Object) save() error {
	db := getDbConnection()
	defer db.Close()

	// TODO handle errors properly
	stmt, err := db.Prepare("insert into objects(url, `interval`) values (?, ?)")
	result, err := stmt.Exec(o.Url, o.Interval)
	o.Id, err = result.LastInsertId()

	return err
}

func (o *Object) addResponse(response string, duration time.Duration) error {
	db := getDbConnection()
	defer db.Close()

	_, err := db.Query(
		"insert into response(object_id, duration, response) values (?, ?, ?)",
		o.Id,
		duration.Seconds(),
		response,
	)

	return err
}

func (o *Object) getHistory() ([]*ObjectHistory, error) {
	// TODO decouple from db dependency
	db := getDbConnection()
	defer db.Close()

	results, err := db.Query("select created_at, duration, response from response where object_id = ?", o.Id)
	var ret []*ObjectHistory

	if nil != err {
		return ret, err
	}

	for results.Next() {
		var oh ObjectHistory
		var createdAt time.Time
		// TODO handle error
		results.Scan(&createdAt, &oh.Duration, &oh.Response)
		oh.CreatedAt = createdAt.Unix()
		ret = append(ret, &oh)
	}

	return ret, nil
}

type ObjectHistory struct {
	Response string `json:"response"`
	Duration float64 `json:"duration"`
	CreatedAt int64 `json:"created_at"`
}

type ObjectHistoryResponse struct {
	*ObjectHistory
}

func (o ObjectHistoryResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type omit bool

type ObjectRequest struct {
	*Object
}

func (or ObjectRequest) Bind(r *http.Request) error {
	o := or.Object
	log.WithFields(log.Fields{"object": o}).Info("Object data received from API")

	if nil == o {
		return errors.New("data incompatible with Object type")
	}

	if o.Url == "" {
		return errors.New("empty URL")
	}

	if u, err := url.Parse(o.Url); err != nil || u.Scheme == "" || u.Host == "" {
		return errors.New(fmt.Sprintf("provided URL \"%s\" is invalid", o.Url))
	}

	return nil
}

type ObjectIdResponse struct {
	*Object
	Url omit `json:"url,omitempty"`
	Interval omit `json:"interval,omitempty"`
	LastCheck omit `json:"last_check,omitempty"`
}

func (o ObjectIdResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type ObjectResponse struct {
	*Object
	LastCheck omit `json:"last_check,omitempty"`
}

func (o ObjectResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// TODO refactor for dependency inversion
func getDbConnection () *sql.DB {
	db, err := sql.Open("mysql", "root:root@tcp(mysql:3306)/object_storage?parseTime=true")

	if err != nil {
		panic("MySQL connection failed")
	}

	return db
}

// TODO refactor this into a repository
func getObjects() []*Object {
	db := getDbConnection()
	defer db.Close()

	// TODO fix error handling
	results, err := db.Query("select o.id, o.url, o.`interval`, max(r.created_at) as last_check " +
		"from objects o " +
		"left join response r on o.id = r.object_id " +
		"group by 1, 2, 3")
	if nil != err {
		panic(err)
	}

	var ret []*Object

	for results.Next() {
		var o Object
		var t sql.NullTime
		// TODO handle error
		results.Scan(&o.Id, &o.Url, &o.Interval, &t)

		if t.Valid {
			o.LastCheck = t.Time
		}

		ret = append(ret, &o)
	}

	return ret
}

type ProgressRegistry struct {
	Ids map[int64]int
	mux sync.Mutex
}

func (p *ProgressRegistry) lock(id int64) {
	p.mux.Lock()
	p.Ids[id] = 1
	p.mux.Unlock()
}

func (p *ProgressRegistry) isLocked(id int64) bool {
	p.mux.Lock()
	_, ok := p.Ids[id]
	defer p.mux.Unlock()

	return ok
}

func (p *ProgressRegistry) unlock(id int64) {
	p.mux.Lock()
	delete(p.Ids, id)
	p.mux.Unlock()
}

// TODO put the worker somewhere separate
func work() {
	var registry = ProgressRegistry{Ids: make(map[int64]int)}

	for {
		// TODO fix this using repository
		for _, o := range getObjects() {
			go handleObject(o, &registry)
		}

		time.Sleep(time.Second)
	}
}

// TODO properly name in a new package and add error handling
func handleObject(o *Object, r *ProgressRegistry) {
	if r.isLocked(o.Id) {
		log.WithFields(log.Fields{"id": o.Id}).
			Debug("Skipping object because it's being processed by another goroutine")

		return
	}

	threshold := o.LastCheck.Add(time.Duration(o.Interval) * time.Second)

	if threshold.Before(time.Now()) {
		r.lock(o.Id)
		log.WithFields(log.Fields{
			"last_check": o.LastCheck,
			"url":        o.Url,
			"interval":   o.Interval,
		}).Info("Interval passed after last check, retrieving response")

		// TODO handle errors
		client := &http.Client{Timeout: 5 * time.Second}
		startTime := time.Now()
		response, _ := client.Get(o.Url)
		duration := time.Now().Sub(startTime)
		body, _ := ioutil.ReadAll(response.Body)
		response.Body.Close()
		bodyString := string(body)

		log.WithFields(log.Fields{"response": bodyString}).Info("Response received")
		if err := o.addResponse(bodyString, duration); nil != err {
			log.WithFields(log.Fields{"error": err}).Error("Error saving response")
		}

		r.unlock(o.Id)
	}
}
