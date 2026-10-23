package health

import (
	"context"

	"github.com/gorilla/mux"

	"github.com/lcnascimento/go-kit/http/httpserver"
)

type Server struct {
	server   *httpserver.Server
	checks   []Check
	shutdown func()
}

func NewServer(port int, shutdown func(), checks ...Check) *Server {
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
	Register(router, s.checks...)

	return nil
}
