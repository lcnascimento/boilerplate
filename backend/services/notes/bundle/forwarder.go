package bundle

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/lcnascimento/go-kit/outbox"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/config"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/worker/forwarder"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

func NewWorkerForwarder() *fx.App {
	return fx.New(workerForwarderOptions()...)
}

func workerForwarderOptions() []fx.Option {
	return []fx.Option{
		Logger(logger),
		Database(),
		fx.Provide(newForwarder, newForwarderHealthServer, newJanitor),
		fx.Invoke(func(lifecycle fx.Lifecycle, f *outbox.Forwarder, janitor *outbox.Janitor, health *forwarder.HealthServer) {
			lifecycle.Append(fx.Hook{OnStart: f.Start, OnStop: f.Stop})
			lifecycle.Append(fx.Hook{OnStart: janitor.Start, OnStop: janitor.Stop})
			lifecycle.Append(fx.Hook{OnStart: health.Start, OnStop: health.Shutdown})
		}),
	}
}

func newForwarder(pool *pgxpool.Pool, shutdowner fx.Shutdowner) (*outbox.Forwarder, error) {
	brokers, err := config.KafkaBrokers()
	if err != nil {
		return nil, err
	}

	return outbox.NewForwarder(notes.Outbox, pool, brokers, []string{events.Topic}, outbox.WithShutdown(exitWithFailure(shutdowner)))
}

func newJanitor(pool *pgxpool.Pool) *outbox.Janitor {
	return outbox.NewJanitor(notes.Outbox, pool)
}

func newForwarderHealthServer(
	db transport.Database, f *outbox.Forwarder, shutdowner fx.Shutdowner,
) (*forwarder.HealthServer, error) {
	port, err := config.HealthPort()
	if err != nil {
		return nil, err
	}

	return forwarder.NewHealthServer(port, db, f, exitWithFailure(shutdowner)), nil
}

func exitWithFailure(shutdowner fx.Shutdowner) func() {
	return func() { _ = shutdowner.Shutdown(fx.ExitCode(1)) }
}
