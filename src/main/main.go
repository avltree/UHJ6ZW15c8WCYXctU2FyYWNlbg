package main

import (
	"../object"
	"../simplerepo"
	"../worker"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
)

var repo object.ObjectRepository = &simplerepo.SimpleRepo{
	Dsn: fmt.Sprintf(
			"%s:%s@tcp(mysql:%s)/%s?parseTime=true",
			os.Getenv("MYSQL_USER"),
			os.Getenv("MYSQL_PASSWORD"),
			os.Getenv("MYSQL_PORT"),
			os.Getenv("MYSQL_DATABASE"),
		),
}

func main() {
	object.Repo = repo
	worker.Repo = repo
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

	worker.Work()
	err := http.ListenAndServe(":8080", r)

	panic(err)
}
