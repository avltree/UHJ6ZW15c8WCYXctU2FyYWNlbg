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

// Repository used by the API and the worker, for more information check the simplerepo.go documentation
var repo object.ObjectRepository = &simplerepo.SimpleRepo{
	Dsn: fmt.Sprintf(
			"%s:%s@tcp(mysql:%s)/%s?parseTime=true",
			os.Getenv("MYSQL_USER"),
			os.Getenv("MYSQL_PASSWORD"),
			os.Getenv("MYSQL_PORT"),
			os.Getenv("MYSQL_DATABASE"),
		),
}

// Main function, starts the HTTP server and the worker
func main() {
	// Assign the repository implementation to our services
	object.Repo = repo
	worker.Repo = repo

	// Set the logger formatting
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	// Setup the router with some middlewares
	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer, middleware.URLFormat)
	r.Use(middleware.Timeout(60))
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(ConstrainPayload)

	// Assign the routes to specific handlers, see handlers.go for implementation
	r.Route("/api/fetcher", func(r chi.Router) {
		r.Get("/", getObjectList)
		r.Post("/", postObject)
		r.Route("/{objectId}", func(r chi.Router) {
			r.Use(getObjectContext)
			r.Delete("/", deleteObject)
			r.Get("/history", getObjectHistory)
		})
	})

	// Start the worker...
	// TODO error handling
	worker.Work()
	// ... and the HTTP server
	err := http.ListenAndServe(":8080", r)

	if nil != err {
		log.WithFields(log.Fields{"error": err}).Error("Could not start the HTTP server")
		panic(err)
	}
}
