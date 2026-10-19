package migrations_test

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/migrations"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/testdb"
)

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

	_, err := db.ExecContext(t.Context(), `INSERT INTO goose_db_version (version_id, is_applied) VALUES (99999, true)`)
	require.NoError(t, err)

	err = migrations.Up(t.Context(), db)

	require.ErrorIs(t, err, migrations.ErrUnknownAppliedVersion)
	require.Contains(t, err.Error(), "make db.reset")
}

func TestComponentMigrations_DownRevertsTheLatestMigration(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrations.Up(t.Context(), db))
	require.NoError(t, migrations.Down(t.Context(), db))

	require.False(t, tableExists(t, db, "watermill_notes_outbox"))
	require.True(t, tableExists(t, db, "notes"))
}

func TestComponentMigrations_EveryMigrationRollsBackAndReapplies(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrations.Up(t.Context(), db))

	for rollbacks := 0; ; rollbacks++ {
		require.Less(t, rollbacks, 1000, "Down never ran out of migrations")

		err := migrations.Down(t.Context(), db)
		if errors.Is(err, migrations.ErrNothingToRollBack) {
			break
		}

		require.NoError(t, err)
	}

	require.False(t, tableExists(t, db, "notes"))
	require.NoError(t, migrations.Up(t.Context(), db))
	require.True(t, tableExists(t, db, "notes"))
}

func TestComponentMigrations_DownWithNothingAppliedFails(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.ErrorIs(t, migrations.Down(t.Context(), db), migrations.ErrNothingToRollBack)
}

func TestComponentMigrations_DownWithAppliedVersionWithoutFileAsksForReset(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrations.Up(t.Context(), db))

	_, err := db.ExecContext(t.Context(), `INSERT INTO goose_db_version (version_id, is_applied) VALUES (99999, true)`)
	require.NoError(t, err)

	err = migrations.Down(t.Context(), db)

	require.ErrorIs(t, err, migrations.ErrUnknownAppliedVersion)
	require.True(t, tableExists(t, db, "notes"))
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()

	var exists bool

	err := db.QueryRowContext(t.Context(), `SELECT to_regclass('public.' || $1) IS NOT NULL`, table).Scan(&exists)
	require.NoError(t, err)

	return exists
}

func newDatabase(t *testing.T) *sql.DB {
	t.Helper()

	if testing.Short() {
		t.Skip("component test")
	}

	return testdb.NewEmpty(t)
}
