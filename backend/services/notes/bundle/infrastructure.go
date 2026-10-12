package bundle

import (
	"context"
	"log/slog"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/lcnascimento/go-kit/env"
	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/validator"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/application"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport"
)

const (
	pkg = "notes/bundle"

	defaultDatabaseURL  = "postgres://notes:notes@localhost:5432/notes?sslmode=disable" //nolint:gosec // local docker-compose credentials
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

// Database opens the service's Postgres pool on DATABASE_URL, traced and
// measured through the global OTel providers, and fails the boot when the
// database does not answer a ping. The pool also backs the /health probe.
func Database() fx.Option {
	return fx.Module(
		"Database",
		fx.Provide(fx.Annotate(newPool, fx.As(fx.Self()), fx.As(new(transport.Database)))),
	)
}

func newPool(lifecycle fx.Lifecycle) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(env.Get("DATABASE_URL", env.WithDefaultValue(defaultDatabaseURL)))
	if err != nil {
		return nil, ErrConnectDatabase.WithCause(err)
	}

	cfg.ConnConfig.Tracer = otelpgx.NewTracer()

	ctx, cancel := context.WithTimeout(context.Background(), databasePingTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, ErrConnectDatabase.WithCause(err)
	}

	if err := otelpgx.RecordStats(pool); err != nil {
		pool.Close()

		return nil, ErrConnectDatabase.WithCause(err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, ErrConnectDatabase.
			WithAttribute("host", cfg.ConnConfig.Host).
			WithAttribute("database", cfg.ConnConfig.Database).
			WithCause(err)
	}

	lifecycle.Append(fx.StopHook(pool.Close))

	return pool, nil
}

func Repositories() fx.Option {
	return fx.Module(
		"Repositories",
		fx.Provide(fx.Annotate(notes.New, fx.As(new(application.NotesRepository)))),
	)
}
