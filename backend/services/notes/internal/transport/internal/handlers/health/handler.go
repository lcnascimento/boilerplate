package health

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/http/httpserver/util"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

const checkTimeout = 2 * events.PingTimeout

var ErrUnhealthy = errors.New("a readiness check failed")

type Check func(ctx context.Context) error

type Health struct {
	checks []Check
}

func Register(router *mux.Router, checks ...Check) {
	h := &Health{checks: checks}

	router.HandleFunc("/ping", h.Ping).Methods(http.MethodGet).Name("Ping")
	router.HandleFunc("/health", h.Health).Methods(http.MethodGet).Name("Health")
}

func (h *Health) Ping(rw http.ResponseWriter, _ *http.Request) {
	util.WriteMessage(rw, http.StatusOK, "pong")
}

func (h *Health) Health(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), checkTimeout)
	defer cancel()

	for _, check := range h.checks {
		if err := check(ctx); err != nil {
			onUnhealthy(r.Context(), err)
			util.WriteMessage(rw, http.StatusServiceUnavailable, "unhealthy")

			return
		}
	}

	util.WriteMessage(rw, http.StatusOK, "healthy")
}
