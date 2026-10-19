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

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/application"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/config"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/repository/entries"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport"
)

const (
	pkg = "audit/bundle"

	databasePingTimeout = 5 * time.Second
)

var ErrConnectDatabase = errors.New("failed to connect to the database: is Postgres up and DATABASE_URL right?")

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

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, ErrConnectDatabase.WithCause(err)
	}

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

func Repositories() fx.Option {
	return fx.Module(
		"Repositories",
		fx.Provide(fx.Annotate(entries.New, fx.As(new(application.AuditEntriesRepository)))),
	)
}
