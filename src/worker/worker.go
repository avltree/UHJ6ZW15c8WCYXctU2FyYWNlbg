package worker

import (
	"../object"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

// Declaration of the repository, using the interface from object package
var Repo object.ObjectRepository

// TODO move the type to a separate file
// Structure used to prevent the worker's goroutines from trying to fetch the response data if another goroutine is
// already processing it. The worker "ticks" take one second, so for example if for some reason fetching of the response
// takes 2 seconds, another one could start doing it, not knowing about the ongoing operation.
// This structure is synchronized between the goroutines and used to "lock" or "unlock" specified objects from being
// processed by goroutines.
type ProgressRegistry struct {
	Ids map[int64]int
	mux sync.Mutex
}

// Locks the object with specified id from processing
func (p *ProgressRegistry) lock(id int64) {
	p.mux.Lock()
	p.Ids[id] = 1
	p.mux.Unlock()
}

// Checks if the object with specified id is locked
func (p *ProgressRegistry) isLocked(id int64) bool {
	p.mux.Lock()
	_, ok := p.Ids[id]
	defer p.mux.Unlock()

	return ok
}

// Unlocks the object for processing
func (p *ProgressRegistry) unlock(id int64) {
	p.mux.Lock()
	delete(p.Ids, id)
	p.mux.Unlock()
}

// Exported method to start the worker. Made this way so it doesn't require explicit goroutine usage
func Work() {
	go execute()
}

// Starts the infinite loop for the worker
func execute()  {
	var registry = ProgressRegistry{Ids: make(map[int64]int)}

	for {
		for _, o := range Repo.FindAll() {
			go handleObject(o, &registry)
		}

		// One-second "ticks"
		time.Sleep(time.Second)
	}
}

// Fetches the response for a single object and stores into the database
func handleObject(o *object.Object, r *ProgressRegistry) {
	if r.isLocked(o.Id) {
		log.WithFields(log.Fields{"id": o.Id}).
			Debug("Skipping object because it's being processed by another goroutine")

		return
	}

	// Last check + [interval] seconds
	threshold := o.LastCheck.Add(time.Duration(o.Interval) * time.Second)

	if threshold.Before(time.Now()) {
		r.lock(o.Id)
		log.WithFields(log.Fields{
			"last_check": o.LastCheck,
			"url":        o.Url,
			"interval":   o.Interval,
		}).Info("Interval passed after last check, retrieving response")

		// TODO handle errors
		// Wait max 5 seconds for a response
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
