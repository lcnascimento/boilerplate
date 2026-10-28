package bundle

import (
	"context"
	"log/slog"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/validator"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/application"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/config"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
)

const (
	pkg = "notes/bundle"

	databasePingTimeout = 5 * time.Second
)

var ErrConnectDatabase = errors.New("failed to connect to the database: is Postgres up and NOTES_DATABASE_URL right?")

func Logger(logger *slog.Logger) fx.Option {
	return fx.Module(
		"Logger",
		fx.Provide(func() *slog.Logger { return logger }),
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logger}
		}),
	)
}

func Validator() fx.Option {
	return fx.Module(
		"Validator",
		fx.Provide(fx.Annotate(
			func() (*validator.Validator, error) {
				return validator.New(validator.WithJSONFieldNames())
			},
			fx.As(new(application.Validator)),
		)),
	)
}

func Database() fx.Option {
	return fx.Module(
		"Database",
		fx.Provide(fx.Annotate(newPool, fx.As(fx.Self()), fx.As(new(transport.Database)))),
	)
}

func newPool(lifecycle fx.Lifecycle) (*pgxpool.Pool, error) {
	url, err := config.DatabaseURL()
	if err != nil {
		return nil, err
	}

	serviceName, err := config.ServiceName()
	if err != nil {
		return nil, err
	}

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, ErrConnectDatabase.WithCause(err)
	}

	cfg.ConnConfig.RuntimeParams["application_name"] = serviceName
	cfg.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, ErrConnectDatabase.WithCause(err)
	}

	if err := otelpgx.RecordStats(pool); err != nil {
		pool.Close()

		return nil, ErrConnectDatabase.WithCause(err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, databasePingTimeout)
			defer cancel()

			if err := pool.Ping(ctx); err != nil {
				return ErrConnectDatabase.
					WithAttribute("host", cfg.ConnConfig.Host).
					WithAttribute("database", cfg.ConnConfig.Database).
					WithCause(err)
			}

			return nil
		},
		OnStop: func(context.Context) error {
			pool.Close()

			return nil
		},
	})

	return pool, nil
}

func ActorMiddleware() fx.Option {
	return fx.Module(
		"Actor Middleware",
		fx.Provide(fx.Annotate(newActorMiddleware, fx.As(new(transport.Authenticator)))),
	)
}

func newActorMiddleware(lifecycle fx.Lifecycle) (*actor.Middleware, error) {
	cfg, err := config.ActorAuth()
	if err != nil {
		return nil, err
	}

	middleware := actor.NewMiddleware(cfg)
	lifecycle.Append(fx.Hook{OnStart: middleware.Start, OnStop: middleware.Stop})

	return middleware, nil
}

func Repositories() fx.Option {
	return fx.Module(
		"Repositories",
		fx.Provide(newSchemaIDs),
		fx.Provide(fx.Annotate(notes.New, fx.As(new(application.NotesRepository)))),
	)
}
