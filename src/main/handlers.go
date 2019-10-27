package main

import (
	"../object"
	"context"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

// Reusable logic to render HTTP errors and log some data if necessary
// FIXME could be reworked not to violate the dependency inversion principle
func logAndRenderError(w http.ResponseWriter, statusCode int, fields log.Fields, message string)  {
	log.WithFields(fields).Error(message)
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(""))
}

// Custom middleware used to return HTTP status code 413 when the request payload exceeds 1MB
// I don't know if it's the right way to do this but I've tried to improvise ;)
func ConstrainPayload(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"actual": r.ContentLength,
			"max": 1 << (10 * 2),
		}).Debug("Content length")

		if r.ContentLength > 1 << (10 * 2) {
			logAndRenderError(w, http.StatusRequestEntityTooLarge, log.Fields{}, "Payload exceeding 1 MB posted")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Handler for the GET /api/fetcher resource
func getObjectList(w http.ResponseWriter, r *http.Request)  {
	var list []render.Renderer
	objects, err := repo.FindAll()

	if nil != err {
		logAndRenderError(w, http.StatusInternalServerError, log.Fields{"error": err}, "Error retrieving objects")
	}

	for _, o := range objects {
		list = append(list, ObjectResponse{Object: o})
	}

	if 0 == len(list) {
		// A workaround for an empty list, because the renderer renders 'null' instead of an empty array
		_, _ = w.Write([]byte("[]"))
		return
	}

	_ = render.RenderList(w, r, list)
}

// Middleware used to load an Object instance from the URL param into the context
// If the specified object isn't found, here a 404 HTTP error is returned, or a 400 - when the URL parameter is invalid
func getObjectContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stringId := chi.URLParam(r, "objectId")
		log.WithFields(log.Fields{"id": stringId}).Info("Object id retrieved from URL")

		if "" == stringId {
			logAndRenderError(w, http.StatusBadRequest, log.Fields{}, "No object id provided")
			return
		}

		id, err := strconv.Atoi(stringId)

		if nil != err || id <= 0 {
			logAndRenderError(w, http.StatusBadRequest, log.Fields{"id": stringId}, "Id is not a positive integer")
			return
		}

		o, err := repo.FindOne(int64(id))

		if nil != err {
			logAndRenderError(w, http.StatusNotFound, log.Fields{
				"id": id,
				"err": err,
			}, "Error searching object in the database")
			return
		}

		ctx := context.WithValue(r.Context(), "object", o)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Handler for the POST api/fetcher resource.
func postObject(w http.ResponseWriter, r *http.Request) {
	data := &ObjectRequest{}

	if err := render.Bind(r, data); err != nil {
		logAndRenderError(w, http.StatusBadRequest, log.Fields{"error": err}, "Error creating new Object")
		return
	}

	o := data.Object

	if err := o.Save(); err != nil {
		logAndRenderError(w, http.StatusInternalServerError, log.Fields{"error": err.Error()}, "Error saving object")
		return
	}

	render.Status(r, http.StatusCreated)
	_ = render.Render(w, r, ObjectIdResponse{
		Object: o,
	})
}

// Handler for the DELETE /api/fetcher/<id> resource.
func deleteObject(w http.ResponseWriter, r *http.Request) {
	o := r.Context().Value("object").(*object.Object)
	log.WithFields(log.Fields{"object": o}).Info("Deleting object")

	if err := o.Delete(); nil != err {
		logAndRenderError(w, http.StatusInternalServerError, log.Fields{"error": err.Error()}, "Error deleting object")
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(""))
	return
}

// Handler for the GET api/fetcher/<id>/history resource.
func getObjectHistory(w http.ResponseWriter, r *http.Request)  {
	o := r.Context().Value("object").(*object.Object)
	log.WithFields(log.Fields{"object": o}).Info("Retrieving object history")

	var list []render.Renderer
	history, err := o.GetHistory()

	if nil != err {
		logAndRenderError(w, http.StatusInternalServerError, log.Fields{"error": err.Error()}, "Error getting object history")
		return
	}

	for _, oh := range history {
		list = append(list, ObjectHistoryResponse{oh})
	}

	if 0 == len(list) {
		// A workaround for an empty list, because the renderer renders 'null' instead of an empty array
		_, _ = w.Write([]byte("[]"))
		return
	}

	_ = render.RenderList(w, r, list)
}
