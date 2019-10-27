package object

import (
	"time"
)

// Declares the interface for object repository so the object is decoupled from the repository implementation
type ObjectRepository interface {
	FindOne(id int64) (*Object, error)
	FindAll() ([]*Object, error)
	Delete(o *Object) error
	Save(o *Object) error
	AddResponse(o *Object, response string, duration time.Duration) error
	GetHistory(o *Object) ([]*ObjectHistory, error)
}

// Declaration of the repository, using the interface from object package
var Repo ObjectRepository

// The Object is the main data structure, represents the data provided by the POST resource
type Object struct {
	Id int64 `json:"id"`
	Url string `json:"url"`
	Interval int `json:"interval"`
	LastCheck time.Time `json:"last_check"`
}

// Deletes the object from the database
func (o *Object) Delete() error {
	return Repo.Delete(o)
}

// Saves the object into the database
func (o *Object) Save() error {
	return Repo.Save(o)
}

// Inserts a new response for the specified object into the database
func (o *Object) AddResponse(response string, duration time.Duration) error {
	return Repo.AddResponse(o, response, duration)
}

// Gets the object history
func (o *Object) GetHistory() ([]*ObjectHistory, error) {
	return Repo.GetHistory(o)
}

// Data structure for object history
type ObjectHistory struct {
	Response string `json:"response"`
	Duration float64 `json:"duration"`
	CreatedAt int64 `json:"created_at"`
}
