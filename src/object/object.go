package object

import (
	"time"
)

type ObjectRepository interface {
	FindOne(id int64) (*Object, error)
	FindAll() []*Object
	Delete(o *Object) error
	Save(o *Object) error
	AddResponse(o *Object, response string, duration time.Duration) error
	GetHistory(o *Object) ([]*ObjectHistory, error)
}

var Repo ObjectRepository

type Object struct {
	Id int64 `json:"id"`
	Url string `json:"url"`
	Interval int `json:"interval"`
	LastCheck time.Time `json:"last_check"`
}

func (o *Object) Delete() error {
	return Repo.Delete(o)
}

func (o *Object) Save() error {
	return Repo.Save(o)
}

func (o *Object) AddResponse(response string, duration time.Duration) error {
	return Repo.AddResponse(o, response, duration)
}

func (o *Object) GetHistory() ([]*ObjectHistory, error) {
	return Repo.GetHistory(o)
}

type ObjectHistory struct {
	Response string `json:"response"`
	Duration float64 `json:"duration"`
	CreatedAt int64 `json:"created_at"`
}
