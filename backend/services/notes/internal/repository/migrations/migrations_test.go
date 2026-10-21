package migrations_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/postgres/migrate"
	"github.com/lcnascimento/go-kit/postgres/testdb"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/migrations"
)

func TestComponentMigrations_CreatesNotesTable(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrate.Up(t.Context(), db, migrations.FS))

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

func TestComponentMigrations_EveryMigrationRollsBackAndReapplies(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrate.Up(t.Context(), db, migrations.FS))

	for rollbacks := 0; ; rollbacks++ {
		require.Less(t, rollbacks, 1000, "Down never ran out of migrations")

		err := migrate.Down(t.Context(), db, migrations.FS)
		if errors.Is(err, migrate.ErrNothingToRollBack) {
			break
		}

		require.NoError(t, err)
	}

	require.False(t, tableExists(t, db, "notes"))
	require.NoError(t, migrate.Up(t.Context(), db, migrations.FS))
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
