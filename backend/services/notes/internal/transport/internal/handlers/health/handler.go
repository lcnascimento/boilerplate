package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/http/httpserver/util"
)

const checkTimeout = 2 * time.Second

var ErrUnhealthy = errors.New("a readiness check failed")

// Check is one readiness condition of /health, such as a database ping.
type Check func(ctx context.Context) error

type Health struct {
	checks []Check
}

// Register serves /ping (liveness) and /health (readiness: every check passes).
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
