package migrations_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/postgres/migrate"
	"github.com/lcnascimento/go-kit/postgres/testdb"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/repository/migrations"
)

func TestComponentMigrations_CreatesAuditEntriesTableStampingReceivedAt(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrate.Up(t.Context(), db, migrations.FS))

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

func TestComponentMigrations_ActorColumnsDefaultToNull(t *testing.T) {
	t.Parallel()

	db := newDatabase(t)

	require.NoError(t, migrate.Up(t.Context(), db, migrations.FS))

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO audit_entries (source, id, type, spec_version, time, data_content_type,
			headers, payload, kafka_topic, kafka_partition, kafka_offset)
		VALUES ('notes', 'event-1', 'boilerplate.notes.v1.NoteDeleted', '1.0', now(), 'application/protobuf',
			'{}', '\x00', 'notes.note.v1', 0, 0)`)
	require.NoError(t, err)

	var actorID, actorOrganizationID sql.NullString

	err = db.QueryRowContext(t.Context(), `SELECT actor_id, actor_organization_id FROM audit_entries`).
		Scan(&actorID, &actorOrganizationID)
	require.NoError(t, err)
	require.False(t, actorID.Valid)
	require.False(t, actorOrganizationID.Valid)
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

	require.False(t, auditEntriesExists(t, db))
	require.NoError(t, migrate.Up(t.Context(), db, migrations.FS))
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
