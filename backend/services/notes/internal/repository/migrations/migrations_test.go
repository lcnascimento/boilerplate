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

const acmeOrganizationID = "9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f"

func TestComponentMigrations_CreatesNotesTable(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrate.Up(t.Context(), db, migrations.FS))

	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)

	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(t.Context(), `SELECT set_config('app.organization_id', $1, true)`, acmeOrganizationID)
	require.NoError(t, err)

	createdAt := time.Date(2026, 10, 12, 18, 0, 0, 0, time.UTC)
	_, err = tx.ExecContext(t.Context(),
		`INSERT INTO notes (id, organization_id, created_at, title, body) VALUES ($1, $2, $3, $4, $5)`,
		"0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e", acmeOrganizationID, createdAt, "Lista de compras", "Leite e pão",
	)
	require.NoError(t, err)

	var organizationID, title, body string

	var gotCreatedAt time.Time

	err = tx.QueryRowContext(t.Context(), `SELECT organization_id, title, body, created_at FROM notes`).
		Scan(&organizationID, &title, &body, &gotCreatedAt)
	require.NoError(t, err)
	require.Equal(t, acmeOrganizationID, organizationID)
	require.Equal(t, "Lista de compras", title)
	require.Equal(t, "Leite e pão", body)
	require.True(t, createdAt.Equal(gotCreatedAt))
}

func TestComponentMigrations_NotesRejectAnotherOrganizationsRow(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrate.Up(t.Context(), db, migrations.FS))

	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)

	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(t.Context(), `SELECT set_config('app.organization_id', $1, true)`, acmeOrganizationID)
	require.NoError(t, err)

	_, err = tx.ExecContext(t.Context(),
		`INSERT INTO notes (id, organization_id, created_at, title, body) VALUES ($1, $2, now(), 'x', '')`,
		"0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e", "4d7a1c3e-5b6f-4a8b-9c0d-1e2f3a4b5c6d",
	)
	require.ErrorContains(t, err, `new row violates row-level security policy for table "notes"`)
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
