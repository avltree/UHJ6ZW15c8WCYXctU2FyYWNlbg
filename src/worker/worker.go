package worker

import (
	"../object"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

var Repo object.ObjectRepository

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

func Work() {
	go execute()
}

func execute()  {
	var registry = ProgressRegistry{Ids: make(map[int64]int)}

	for {
		// TODO fix this using repository
		for _, o := range Repo.FindAll() {
			go handleObject(o, &registry)
		}

		time.Sleep(time.Second)
	}
}

func handleObject(o *object.Object, r *ProgressRegistry) {
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
		if err := o.AddResponse(bodyString, duration); nil != err {
			log.WithFields(log.Fields{"error": err}).Error("Error saving response")
		}

		r.unlock(o.Id)
	}
}
