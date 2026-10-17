package routes

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/api/frontend/internal/handlers/health"
)

func RegisterHealth(router *mux.Router, db transport.Database) {
	handler := health.New(db)

	router.HandleFunc("/ping", handler.Ping).Methods(http.MethodGet).Name("Ping")
	router.HandleFunc("/health", handler.Health).Methods(http.MethodGet).Name("Health")
}
