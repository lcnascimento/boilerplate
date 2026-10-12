// Package migrations holds the service's forward-only SQL migrations and
// applies them with goose.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"slices"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/lock"

	"github.com/lcnascimento/go-kit/errors"
)

//go:embed *.sql
var files embed.FS

var (
	ErrUnknownAppliedVersion = errors.New(
		"the database has migrations applied that this checkout does not have (switched branches?): run `make db.reset`",
	).WithCode("UNKNOWN_APPLIED_MIGRATION")

	ErrMigrate = errors.New("failed to apply migrations")
)

// Up applies every pending migration, each in its own transaction, under a
// Postgres advisory lock so concurrent runs wait for each other. It refuses to
// run when the database holds a version with no matching file: goose v3 skips
// those silently, which would leave a branch's schema behind unnoticed.
func Up(ctx context.Context, db *sql.DB) error {
	store, err := database.NewStore(database.DialectPostgres, goose.DefaultTablename)
	if err != nil {
		return ErrMigrate.WithCause(err)
	}

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return ErrMigrate.WithCause(err)
	}

	provider, err := goose.NewProvider("", db, files, goose.WithStore(store), goose.WithSessionLocker(locker))
	if err != nil {
		return ErrMigrate.WithCause(err)
	}

	// Returned unwrapped: its message is the one thing the developer must read.
	if err := checkAppliedVersionsAreKnown(ctx, db, store, provider.ListSources()); err != nil {
		return err
	}

	if _, err := provider.Up(ctx); err != nil {
		return ErrMigrate.WithCause(err)
	}

	return nil
}

func checkAppliedVersionsAreKnown(ctx context.Context, db *sql.DB, store database.Store, sources []*goose.Source) error {
	extender, ok := store.(database.StoreExtender)
	if !ok {
		return ErrMigrate.WithAttribute("reason", "goose store cannot check whether its table exists")
	}

	exists, err := extender.TableExists(ctx, db)
	if err != nil {
		return ErrMigrate.WithCause(err)
	}

	if !exists {
		return nil
	}

	applied, err := store.ListMigrations(ctx, db)
	if err != nil {
		return ErrMigrate.WithCause(err)
	}

	var unknown []int64

	for _, m := range applied {
		// Version 0 is the row goose inserts when it creates its table.
		known := m.Version == 0 || slices.ContainsFunc(sources, func(s *goose.Source) bool { return s.Version == m.Version })
		if m.IsApplied && !known {
			unknown = append(unknown, m.Version)
		}
	}

	if len(unknown) > 0 {
		return ErrUnknownAppliedVersion.WithAttribute("versions", fmt.Sprint(unknown))
	}

	return nil
}
