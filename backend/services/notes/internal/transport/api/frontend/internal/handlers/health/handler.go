package health

import (
	"context"
	"net/http"
	"time"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/http/httpserver/util"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
)

// pingTimeout bounds the database ping, so a slow database answers the probe
// with 503 instead of hanging it.
const pingTimeout = 2 * time.Second

var ErrDatabaseUnhealthy = errors.New("database did not answer the health ping")

type Health struct {
	db transport.Database
}

func New(db transport.Database) *Health {
	return &Health{db: db}
}

// Ping is liveness: it answers while the process does, whatever the database says.
func (h *Health) Ping(rw http.ResponseWriter, _ *http.Request) {
	util.WriteMessage(rw, http.StatusOK, "pong")
}

// Health is readiness: it reflects whether the database answers.
func (h *Health) Health(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), pingTimeout)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		onUnhealthy(r.Context(), err)
		util.WriteMessage(rw, http.StatusServiceUnavailable, "unhealthy")

		return
	}

	util.WriteMessage(rw, http.StatusOK, "healthy")
}
