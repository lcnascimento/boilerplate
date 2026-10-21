package entries_test

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/postgres/testdb"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/idomain"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/repository/entries"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/repository/migrations"
)

type row struct {
	Source          string
	ID              string
	Type            string
	SpecVersion     string
	Subject         *string
	Time            time.Time
	DataContentType string
	Headers         map[string]string
	Payload         []byte
	KafkaTopic      string
	KafkaPartition  int32
	KafkaOffset     int64
}

func TestComponentRepository_InsertStoresEveryField(t *testing.T) {
	t.Parallel()

	pool := newPool(t)
	repo := entries.New(pool)

	require.NoError(t, repo.Insert(t.Context(), noteCreatedEntry()))

	rows := readRows(t, pool)
	require.Len(t, rows, 1)
	require.Equal(t, row{
		Source:          "notes",
		ID:              "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
		Type:            "boilerplate.notes.v1.NoteCreated",
		SpecVersion:     "1.0",
		Subject:         new("note_0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b00"),
		Time:            time.Date(2026, 10, 17, 12, 30, 0, 123456000, time.UTC),
		DataContentType: "application/protobuf",
		Headers: map[string]string{
			"ce_id":     "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
			"ce_source": "notes",
		},
		Payload:        []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x0a, 0x02, 'h', 'i'},
		KafkaTopic:     "notes.note.v1",
		KafkaPartition: 2,
		KafkaOffset:    41,
	}, rows[0])
}

func TestComponentRepository_InsertWithoutSubjectStoresNull(t *testing.T) {
	t.Parallel()

	pool := newPool(t)
	entry := noteCreatedEntry()
	entry.Subject = ""

	require.NoError(t, entries.New(pool).Insert(t.Context(), entry))

	rows := readRows(t, pool)
	require.Len(t, rows, 1)
	require.Nil(t, rows[0].Subject)
}

func TestComponentRepository_InsertOfSameSourceAndIDKeepsTheOriginal(t *testing.T) {
	t.Parallel()

	pool := newPool(t)
	repo := entries.New(pool)

	require.NoError(t, repo.Insert(t.Context(), noteCreatedEntry()))

	redelivered := noteCreatedEntry()
	redelivered.Payload = []byte("other")
	redelivered.KafkaPartition = 0
	redelivered.KafkaOffset = 99

	require.NoError(t, repo.Insert(t.Context(), redelivered))

	rows := readRows(t, pool)
	require.Len(t, rows, 1)
	require.Equal(t, []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x0a, 0x02, 'h', 'i'}, rows[0].Payload)
	require.Equal(t, int32(2), rows[0].KafkaPartition)
	require.Equal(t, int64(41), rows[0].KafkaOffset)
}

func TestComponentRepository_InsertOfSameIDFromAnotherSourceIsAnotherEntry(t *testing.T) {
	t.Parallel()

	pool := newPool(t)
	repo := entries.New(pool)

	require.NoError(t, repo.Insert(t.Context(), noteCreatedEntry()))

	other := noteCreatedEntry()
	other.Source = "billing"

	require.NoError(t, repo.Insert(t.Context(), other))

	require.Len(t, readRows(t, pool), 2)
}

func TestComponentRepository_InsertFailsWhenTheDatabaseIsClosed(t *testing.T) {
	t.Parallel()

	pool := newPool(t)
	pool.Close()

	err := entries.New(pool).Insert(t.Context(), noteCreatedEntry())

	require.ErrorIs(t, err, entries.ErrQuery)
}

func noteCreatedEntry() *idomain.Entry {
	return &idomain.Entry{
		Source:          "notes",
		ID:              "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
		Type:            "boilerplate.notes.v1.NoteCreated",
		SpecVersion:     "1.0",
		Subject:         "note_0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b00",
		Time:            time.Date(2026, 10, 17, 12, 30, 0, 123456000, time.UTC),
		DataContentType: "application/protobuf",
		Headers: map[string]string{
			"ce_id":     "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
			"ce_source": "notes",
		},
		Payload:        []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x0a, 0x02, 'h', 'i'},
		KafkaTopic:     "notes.note.v1",
		KafkaPartition: 2,
		KafkaOffset:    41,
	}
}

func readRows(t *testing.T, pool *pgxpool.Pool) []row {
	t.Helper()

	rows, err := pool.Query(t.Context(), `
		SELECT source, id, type, spec_version, subject, time, data_content_type,
			headers, payload, kafka_topic, kafka_partition, kafka_offset
		FROM audit_entries ORDER BY source, id`)
	require.NoError(t, err)

	defer rows.Close()

	var out []row

	for rows.Next() {
		var r row

		require.NoError(t, rows.Scan(
			&r.Source, &r.ID, &r.Type, &r.SpecVersion, &r.Subject, &r.Time, &r.DataContentType,
			&r.Headers, &r.Payload, &r.KafkaTopic, &r.KafkaPartition, &r.KafkaOffset,
		))

		r.Time = r.Time.UTC()
		out = append(out, r)
	}

	require.NoError(t, rows.Err())

	return out
}

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if testing.Short() {
		t.Skip("component test")
	}

	return testdb.New(t, migrations.FS)
}
