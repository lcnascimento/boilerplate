package bundle

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/lcnascimento/go-kit/outbox"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/health"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/config"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes"
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
		fx.Provide(newForwarder, newForwarderHealthServer, newJanitor),
		fx.Invoke(func(lifecycle fx.Lifecycle, f *outbox.Forwarder, janitor *outbox.Janitor, health *health.Server) {
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

	return forwarder.New(context.Background(), notes.Outbox, pool, brokers, exitWithFailure(shutdowner))
}

func newJanitor(pool *pgxpool.Pool) (*outbox.Janitor, error) {
	return outbox.NewJanitor(notes.Outbox, pool)
}

func newForwarderHealthServer(
	db transport.Database, f *outbox.Forwarder, shutdowner fx.Shutdowner,
) (*health.Server, error) {
	port, err := config.HealthPort()
	if err != nil {
		return nil, err
	}

	return health.NewServer(port, exitWithFailure(shutdowner), db.Ping, f.Ready), nil
}

func exitWithFailure(shutdowner fx.Shutdowner) func() {
	return func() { _ = shutdowner.Shutdown(fx.ExitCode(1)) }
}
