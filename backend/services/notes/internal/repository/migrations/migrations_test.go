package migrations_test

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/env"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/migrations"
)

const defaultAdminURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" //nolint:gosec // local docker-compose credentials

func TestComponentMigrations_CreatesNotesTable(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrations.Up(t.Context(), db))

	createdAt := time.Date(2026, 10, 12, 18, 0, 0, 0, time.UTC)
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO notes (id, created_at, title, body) VALUES ($1, $2, $3, $4)`,
		"0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e", createdAt, "Lista de compras", "Leite e pão",
	)
	require.NoError(t, err)

	var title, body string

	var gotCreatedAt time.Time

	err = db.QueryRowContext(t.Context(), `SELECT title, body, created_at FROM notes`).Scan(&title, &body, &gotCreatedAt)
	require.NoError(t, err)
	require.Equal(t, "Lista de compras", title)
	require.Equal(t, "Leite e pão", body)
	require.True(t, createdAt.Equal(gotCreatedAt))
}

func TestComponentMigrations_RerunIsNoop(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrations.Up(t.Context(), db))
	require.NoError(t, migrations.Up(t.Context(), db))
}

func TestComponentMigrations_ConcurrentRunsAllSucceed(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	const runs = 4

	errs := make([]error, runs)

	var wg sync.WaitGroup
	for i := range runs {
		wg.Go(func() { errs[i] = migrations.Up(t.Context(), db) })
	}

	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
}

func TestComponentMigrations_AppliedVersionWithoutFileAsksForReset(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrations.Up(t.Context(), db))

	// Another branch applied a migration this checkout does not have.
	_, err := db.ExecContext(t.Context(), `INSERT INTO goose_db_version (version_id, is_applied) VALUES (99999, true)`)
	require.NoError(t, err)

	err = migrations.Up(t.Context(), db)

	require.ErrorIs(t, err, migrations.ErrUnknownAppliedVersion)
	require.Contains(t, err.Error(), "make db.reset")
}

// newDatabase creates an empty database for one test and drops it at cleanup.
func newDatabase(t *testing.T) *sql.DB {
	t.Helper()

	if testing.Short() {
		t.Skip("component test")
	}

	adminURL := env.Get("POSTGRES_ADMIN_URL", env.WithDefaultValue(defaultAdminURL))

	admin, err := sql.Open("pgx", adminURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = admin.Close() })

	if err := admin.PingContext(t.Context()); err != nil {
		t.Fatalf("postgres is not reachable at POSTGRES_ADMIN_URL (%v): run `tilt up` or `docker compose up -d postgres` in backend/, "+
			"or `make test.unit` to skip component tests", err)
	}

	name := "test_migrations_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.ExecContext(t.Context(), "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	require.NoError(t, err)

	cfg, err := pgx.ParseConfig(adminURL)
	require.NoError(t, err)

	cfg.Database = name

	db := stdlib.OpenDB(*cfg)

	t.Cleanup(func() {
		_ = db.Close()
		ctx := context.WithoutCancel(t.Context())
		_, _ = admin.ExecContext(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
	})

	return db
}
