package forwarder

import (
	"context"

	"github.com/gorilla/mux"

	"github.com/lcnascimento/go-kit/http/httpserver"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/internal/handlers/health"
)

type HealthServer struct {
	server    *httpserver.Server
	db        transport.Database
	forwarder *Forwarder
	shutdown  func()
}

func NewHealthServer(port int, db transport.Database, forwarder *Forwarder, shutdown func()) *HealthServer {
	return &HealthServer{
		server:    httpserver.NewServer(httpserver.WithPort(port)),
		db:        db,
		forwarder: forwarder,
		shutdown:  shutdown,
	}
}

func (s *HealthServer) Start(_ context.Context) error {
	go func() {
		if err := s.server.Start(s.register); err != nil {
			s.shutdown()
		}
	}()

	return nil
}

func (s *HealthServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *HealthServer) register(router *mux.Router) error {
	health.Register(router, s.db.Ping, s.forwarder.Ready)

	return nil
}
