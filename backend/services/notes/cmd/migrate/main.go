// Command migrate applies the service's migrations to DATABASE_URL and exits.
// It skips the bundle on purpose: no fx and no OTel, only slog on stderr.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/lcnascimento/go-kit/env"
	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/migrations"
)

const (
	defaultDatabaseURL = "postgres://notes:notes@localhost:5432/notes?sslmode=disable" //nolint:gosec // local docker-compose credentials
	timeout            = 5 * time.Minute
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	if err := migrate(); err != nil {
		logger.Error(err.Error(), "attributes", errors.Attributes(err), "reasons", errors.Reasons(err))
		os.Exit(1)
	}

	logger.Info("migrations applied")
}

func migrate() error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	db, err := sql.Open("pgx", env.Get("DATABASE_URL", env.WithDefaultValue(defaultDatabaseURL)))
	if err != nil {
		return migrations.ErrMigrate.WithCause(err)
	}

	defer func() { _ = db.Close() }()

	return migrations.Up(ctx, db)
}
