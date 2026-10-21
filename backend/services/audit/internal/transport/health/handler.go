package health

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/http/httpserver/util"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport"
)

const checkTimeout = 2 * events.PingTimeout

var (
	ErrDatabaseUnhealthy = errors.New("database did not answer the health ping")
	ErrConsumerNotReady  = errors.New("events consumer is not consuming or cannot reach Kafka")
)

type handler struct {
	db       transport.Database
	consumer transport.Consumer
}

func Register(router *mux.Router, db transport.Database, consumer transport.Consumer) {
	h := &handler{db: db, consumer: consumer}

	router.HandleFunc("/ping", h.ping).Methods(http.MethodGet).Name("Ping")
	router.HandleFunc("/health", h.health).Methods(http.MethodGet).Name("Health")
}

func (h *handler) ping(rw http.ResponseWriter, _ *http.Request) {
	util.WriteMessage(rw, http.StatusOK, "pong")
}

func (h *handler) health(rw http.ResponseWriter, r *http.Request) {
	if err := h.check(r.Context()); err != nil {
		onUnhealthy(r.Context(), err)
		util.WriteMessage(rw, http.StatusServiceUnavailable, "unhealthy")

		return
	}

	util.WriteMessage(rw, http.StatusOK, "healthy")
}

func (h *handler) check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	if err := h.consumer.Ready(ctx); err != nil {
		return ErrConsumerNotReady.WithCause(err)
	}

	if err := h.db.Ping(ctx); err != nil {
		return ErrDatabaseUnhealthy.WithCause(err)
	}

	return nil
}
