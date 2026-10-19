package health

import (
	"context"

	"github.com/gorilla/mux"

	"github.com/lcnascimento/go-kit/http/httpserver"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport"
)

type Server struct {
	db       transport.Database
	consumer transport.Consumer
	server   *httpserver.Server
	shutdown func()
}

func New(db transport.Database, consumer transport.Consumer, port int, shutdown func()) *Server {
	return &Server{
		db:       db,
		consumer: consumer,
		server:   httpserver.NewServer(httpserver.WithPort(port)),
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
	Register(router, s.db, s.consumer)

	return nil
}
