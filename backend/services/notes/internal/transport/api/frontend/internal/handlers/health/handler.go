package health

import (
	"net/http"

	"github.com/lcnascimento/go-kit/http/httpserver/util"
)

type Health struct{}

func New() *Health {
	return &Health{}
}

func (h *Health) Ping(rw http.ResponseWriter, _ *http.Request) {
	util.WriteMessage(rw, http.StatusOK, "pong")
}

// Health has no dependency to probe: the repository lives in memory.
func (h *Health) Health(rw http.ResponseWriter, _ *http.Request) {
	util.WriteMessage(rw, http.StatusOK, "healthy")
}
