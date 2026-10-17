package migrations_test

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/repository/internal/testdb"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/repository/migrations"
)

func TestComponentMigrations_CreatesAuditEntriesTableStampingReceivedAt(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrations.Up(t.Context(), db))

	before := time.Now()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO audit_entries (source, id, type, spec_version, time, data_content_type,
			headers, payload, kafka_topic, kafka_partition, kafka_offset)
		VALUES ('notes', 'event-1', 'boilerplate.notes.v1.NoteDeleted', '1.0', now(), 'application/protobuf',
			'{}', '\x00', 'notes.note.v1', 0, 0)`)
	require.NoError(t, err)

	var subject sql.NullString

	var receivedAt time.Time

	err = db.QueryRowContext(t.Context(), `SELECT subject, received_at FROM audit_entries`).Scan(&subject, &receivedAt)
	require.NoError(t, err)
	require.False(t, subject.Valid)
	require.WithinDuration(t, before, receivedAt, time.Minute)
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

	require.False(t, auditEntriesExists(t, db))
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

	require.False(t, auditEntriesExists(t, db))
	require.NoError(t, migrations.Up(t.Context(), db))
	require.True(t, auditEntriesExists(t, db))
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
	require.True(t, auditEntriesExists(t, db))
}

func auditEntriesExists(t *testing.T, db *sql.DB) bool {
	t.Helper()

	var exists bool

	err := db.QueryRowContext(t.Context(), `SELECT to_regclass('public.audit_entries') IS NOT NULL`).Scan(&exists)
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
