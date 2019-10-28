package worker

import (
	"../object"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

// Declaration of the repository, using the interface from object package
var Repo object.ObjectRepository

// Exported method to start the worker. Made this way so it doesn't require explicit goroutine usage
func Work() {
	go execute()
}

// Starts the infinite loop for the worker
func execute() {
	var registry = ProgressRegistry{Ids: make(map[int64]int)}

	for {
		objects, err := Repo.FindAll()

		if nil != err {
			log.WithFields(log.Fields{"error": err}).Error("Could not retrieve objects from the database")
		}

		for _, o := range objects {
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

		// Wait max 5 seconds for a response
		client := &http.Client{Timeout: 5 * time.Second}
		startTime := time.Now()
		response, err := client.Get(o.Url)

		if nil != err {
			log.WithFields(log.Fields{"error": err}).Error("Error fetching response")
			r.unlock(o.Id)
			return
		}

		duration := time.Now().Sub(startTime)
		body, err := ioutil.ReadAll(response.Body)
		response.Body.Close()

		if nil != err {
			log.WithFields(log.Fields{"error": err}).Error("Response body could not be read")
			r.unlock(o.Id)
			return
		}

		bodyString := string(body)
		log.WithFields(log.Fields{"response": bodyString}).Info("Response received")

		if err := o.AddResponse(bodyString, duration); nil != err {
			log.WithFields(log.Fields{"error": err}).Error("Error saving response")
		}

		r.unlock(o.Id)
	}
}
