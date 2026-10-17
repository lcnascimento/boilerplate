package bundle

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/config"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/outbox"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/worker/forwarder"
)

func NewWorkerForwarder() *fx.App {
	return fx.New(workerForwarderOptions()...)
}

func workerForwarderOptions() []fx.Option {
	return []fx.Option{
		Logger(logger),
		Database(),
		fx.Provide(newForwarder, newForwarderHealthServer),
		fx.Invoke(func(lifecycle fx.Lifecycle, f *forwarder.Forwarder, health *forwarder.HealthServer) {
			lifecycle.Append(fx.Hook{OnStart: f.Start, OnStop: f.Stop})
			lifecycle.Append(fx.Hook{OnStart: health.Start, OnStop: health.Shutdown})
		}),
	}
}

func newForwarder(pool *pgxpool.Pool, shutdowner fx.Shutdowner) (*forwarder.Forwarder, error) {
	brokers, err := config.KafkaBrokers()
	if err != nil {
		return nil, err
	}

	subscriber, err := outbox.NewSubscriber(pool)
	if err != nil {
		return nil, err
	}

	return forwarder.New(subscriber, brokers, forwarder.WithShutdown(func() {
		_ = shutdowner.Shutdown(fx.ExitCode(1))
	}))
}

func newForwarderHealthServer(db transport.Database, f *forwarder.Forwarder) (*forwarder.HealthServer, error) {
	port, err := config.HealthPort()
	if err != nil {
		return nil, err
	}

	return forwarder.NewHealthServer(port, db, f), nil
}
