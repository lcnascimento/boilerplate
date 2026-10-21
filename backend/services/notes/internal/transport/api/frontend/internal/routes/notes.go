package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
	v1 "github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/api/frontend/internal/handlers/notes/v1"
)

func RegisterNotes(router *mux.Router, service transport.NotesService, authenticate mux.MiddlewareFunc) {
	sub := router.PathPrefix("/notes/v1/notes").Subrouter()
	sub.Use(authenticate)

	handler := v1.New(service)

	sub.HandleFunc("", handler.Create).Methods(http.MethodPost).Name("CreateNote")
	sub.HandleFunc("", handler.List).Methods(http.MethodGet).Name("ListNotes")
	sub.HandleFunc("/{id}", handler.Get).Methods(http.MethodGet).Name("GetNote")
	sub.HandleFunc("/{id}", handler.Delete).Methods(http.MethodDelete).Name("DeleteNote")
}
