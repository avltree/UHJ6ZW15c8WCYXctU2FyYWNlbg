package main

import (
	"../object"
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"strconv"
)

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

type omit bool

type ObjectResponse struct {
	*object.Object
	LastCheck omit `json:"last_check,omitempty"`
}

func (o ObjectResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func getObjectList(w http.ResponseWriter, r *http.Request)  {
	var list []render.Renderer

	for _, o := range repo.FindAll() {
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

		o, err := repo.FindOne(int64(id))

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

type ObjectRequest struct {
	*object.Object
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
	*object.Object
	Url omit `json:"url,omitempty"`
	Interval omit `json:"interval,omitempty"`
	LastCheck omit `json:"last_check,omitempty"`
}

func (o ObjectIdResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
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

	if err := o.Save(); err != nil {
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
	o := r.Context().Value("object").(*object.Object)
	log.WithFields(log.Fields{"object": o}).Info("Deleting object")

	if err := o.Delete(); nil != err {
		log.WithFields(log.Fields{"error": err.Error()}).Error("Error deleting object")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(""))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(""))
	return
}

type ObjectHistoryResponse struct {
	*object.ObjectHistory
}

func (o ObjectHistoryResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func getObjectHistory(w http.ResponseWriter, r *http.Request)  {
	o := r.Context().Value("object").(*object.Object)
	log.WithFields(log.Fields{"object": o}).Info("Retrieving object history")

	var list []render.Renderer
	history, err := o.GetHistory()

	if nil != err {
		log.WithFields(log.Fields{"error": err.Error()}).Error("Error getting object history")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(""))
		return
	}

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
