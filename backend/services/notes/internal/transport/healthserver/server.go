package healthserver

import (
	"context"

	"github.com/gorilla/mux"

	"github.com/lcnascimento/go-kit/http/httpserver"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/internal/handlers/health"
)

type Server struct {
	server   *httpserver.Server
	checks   []health.Check
	shutdown func()
}

func New(port int, shutdown func(), checks ...health.Check) *Server {
	return &Server{
		server:   httpserver.NewServer(httpserver.WithPort(port)),
		checks:   checks,
		shutdown: shutdown,
	}
}

func (s *Server) Start(_ context.Context) error {
	go func() {
		if err := s.server.Start(s.register); err != nil {
			s.shutdown()
		}
	}()

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) register(router *mux.Router) error {
	health.Register(router, s.checks...)

	return nil
}
