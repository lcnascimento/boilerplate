package bundle

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/lcnascimento/go-kit/outbox"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/application"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/application/changes"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/config"
	changesrepo "github.com/lcnascimento/boilerplate/backend/services/iam/internal/repository/changes"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/transport"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/transport/worker/forwarder"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/transport/worker/translator"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/health"
)

func NewWorker() *fx.App {
	return fx.New(workerOptions()...)
}

func workerOptions() []fx.Option {
	return []fx.Option{
		Logger(logger),
		Database(),
		fx.Provide(
			newSchemaIDs,
			fx.Annotate(changesrepo.New, fx.As(new(application.ChangesRepository))),
			fx.Annotate(changes.New, fx.As(new(transport.ChangesService))),
			func(svc transport.ChangesService) *translator.Translator { return translator.New(svc) },
			newForwarder,
			newJanitor,
			newHealthServer,
		),
		fx.Invoke(func(
			lifecycle fx.Lifecycle,
			t *translator.Translator,
			f *outbox.Forwarder,
			janitor *outbox.Janitor,
			server *health.Server,
		) {
			lifecycle.Append(fx.Hook{OnStart: t.Start, OnStop: t.Stop})
			lifecycle.Append(fx.Hook{OnStart: f.Start, OnStop: f.Stop})
			lifecycle.Append(fx.Hook{OnStart: janitor.Start, OnStop: janitor.Stop})
			lifecycle.Append(fx.Hook{OnStart: server.Start, OnStop: server.Shutdown})
		}),
	}
}

func newForwarder(pool *pgxpool.Pool, shutdowner fx.Shutdowner) (*outbox.Forwarder, error) {
	brokers, err := config.KafkaBrokers()
	if err != nil {
		return nil, err
	}

	return forwarder.New(context.Background(), changesrepo.Outbox, pool, brokers, exitWithFailure(shutdowner))
}

func newJanitor(pool *pgxpool.Pool) (*outbox.Janitor, error) {
	return outbox.NewJanitor(changesrepo.Outbox, pool)
}

func newHealthServer(db transport.Database, f *outbox.Forwarder, shutdowner fx.Shutdowner) (*health.Server, error) {
	port, err := config.HealthPort()
	if err != nil {
		return nil, err
	}

	return health.NewServer(port, exitWithFailure(shutdowner), db.Ping, f.Ready), nil
}

func exitWithFailure(shutdowner fx.Shutdowner) func() {
	return func() { _ = shutdowner.Shutdown(fx.ExitCode(1)) }
}
