package frontend

import (
	"context"

	"github.com/gorilla/mux"

	"github.com/lcnascimento/go-kit/http/httpserver"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/api/frontend/internal/routes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/internal/handlers/health"
)

type Server struct {
	notes    transport.NotesService
	db       transport.Database
	server   *httpserver.Server
	shutdown func()
}

func New(notes transport.NotesService, db transport.Database, opts ...Option) *Server {
	server := &Server{
		notes:    notes,
		db:       db,
		server:   httpserver.NewServer(),
		shutdown: func() {},
	}

	for _, opt := range opts {
		opt(server)
	}

	return server
}

func (s *Server) Start(_ context.Context) error {
	// httpserver.Server.Start blocks until the listener closes.
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
	health.Register(router, s.db.Ping)
	routes.RegisterNotes(router, s.notes)

	return nil
}
