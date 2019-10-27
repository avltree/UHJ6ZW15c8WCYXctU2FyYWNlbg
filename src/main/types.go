package main

import (
	"../object"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
)

// Used to hide JSON fields we don't want to see in our responses
type omit bool

// Custom response object, used in the object list to omit the `last_check` field
type ObjectResponse struct {
	*object.Object
	LastCheck omit `json:"last_check,omitempty"`
}

func (o ObjectResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// Payload for the Object instance, used mostly to validate the POST data
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

// Another custom response for Object, this one strips fields other than the id.
type ObjectIdResponse struct {
	*object.Object
	Url omit `json:"url,omitempty"`
	Interval omit `json:"interval,omitempty"`
	LastCheck omit `json:"last_check,omitempty"`
}

func (o ObjectIdResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// Custom response object for the ObjectHistory instances.
type ObjectHistoryResponse struct {
	*object.ObjectHistory
}

func (o ObjectHistoryResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
