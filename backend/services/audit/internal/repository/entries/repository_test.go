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

	ActorID             *string
	ActorOrganizationID *string
	IPHash              *string
	CorrelationID       *string
}

func TestComponentRepository_InsertStoresEveryField(t *testing.T) {
	t.Parallel()

	pool := newPool(t)
	repo := entries.New(pool)

	require.NoError(t, repo.Insert(t.Context(), userCreatedEntry()))

	rows := readRows(t, pool)
	require.Len(t, rows, 1)
	require.Equal(t, row{
		Source:          "iam",
		ID:              "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
		Type:            "boilerplate.iam.v1.UserCreated",
		SpecVersion:     "1.0",
		Subject:         new("user_0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b00"),
		Time:            time.Date(2026, 10, 17, 12, 30, 0, 123456000, time.UTC),
		DataContentType: "application/protobuf",
		Headers: map[string]string{
			"ce_id":     "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
			"ce_source": "iam",
		},
		Payload:        []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x0a, 0x02, 'h', 'i'},
		KafkaTopic:     "iam.user.v1",
		KafkaPartition: 2,
		KafkaOffset:    41,

		ActorID:             new("user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b"),
		ActorOrganizationID: new("org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f"),
		IPHash:              new("5d41402abc4b2a76b9719d911017c592ae0b6a0f7d5b3e1c2a4f6e8d0b1c3a5f"),
		CorrelationID:       new("784512"),
	}, rows[0])
}

func TestComponentRepository_InsertWithoutSubjectStoresNull(t *testing.T) {
	t.Parallel()

	pool := newPool(t)
	entry := userCreatedEntry()
	entry.Subject = ""

	require.NoError(t, entries.New(pool).Insert(t.Context(), entry))

	rows := readRows(t, pool)
	require.Len(t, rows, 1)
	require.Nil(t, rows[0].Subject)
}

func TestComponentRepository_InsertWithoutActorIPHashOrCorrelationStoresNull(t *testing.T) {
	t.Parallel()

	pool := newPool(t)
	entry := userCreatedEntry()
	entry.ActorID = ""
	entry.ActorOrganizationID = ""
	entry.IPHash = ""
	entry.CorrelationID = ""

	require.NoError(t, entries.New(pool).Insert(t.Context(), entry))

	rows := readRows(t, pool)
	require.Len(t, rows, 1)
	require.Nil(t, rows[0].ActorID)
	require.Nil(t, rows[0].ActorOrganizationID)
	require.Nil(t, rows[0].IPHash)
	require.Nil(t, rows[0].CorrelationID)
}

func TestComponentRepository_InsertOfSameSourceAndIDKeepsTheOriginal(t *testing.T) {
	t.Parallel()

	pool := newPool(t)
	repo := entries.New(pool)

	require.NoError(t, repo.Insert(t.Context(), userCreatedEntry()))

	redelivered := userCreatedEntry()
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

	require.NoError(t, repo.Insert(t.Context(), userCreatedEntry()))

	other := userCreatedEntry()
	other.Source = "billing"

	require.NoError(t, repo.Insert(t.Context(), other))

	require.Len(t, readRows(t, pool), 2)
}

func TestComponentRepository_InsertFailsWhenTheDatabaseIsClosed(t *testing.T) {
	t.Parallel()

	pool := newPool(t)
	pool.Close()

	err := entries.New(pool).Insert(t.Context(), userCreatedEntry())

	require.ErrorIs(t, err, entries.ErrQuery)
}

func userCreatedEntry() *idomain.Entry {
	return &idomain.Entry{
		Source:          "iam",
		ID:              "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
		Type:            "boilerplate.iam.v1.UserCreated",
		SpecVersion:     "1.0",
		Subject:         "user_0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b00",
		Time:            time.Date(2026, 10, 17, 12, 30, 0, 123456000, time.UTC),
		DataContentType: "application/protobuf",
		Headers: map[string]string{
			"ce_id":     "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
			"ce_source": "iam",
		},
		Payload:        []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x0a, 0x02, 'h', 'i'},
		KafkaTopic:     "iam.user.v1",
		KafkaPartition: 2,
		KafkaOffset:    41,

		ActorID:             "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
		ActorOrganizationID: "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f",
		IPHash:              "5d41402abc4b2a76b9719d911017c592ae0b6a0f7d5b3e1c2a4f6e8d0b1c3a5f",
		CorrelationID:       "784512",
	}
}

func readRows(t *testing.T, pool *pgxpool.Pool) []row {
	t.Helper()

	rows, err := pool.Query(t.Context(), `
		SELECT source, id, type, spec_version, subject, time, data_content_type,
			headers, payload, kafka_topic, kafka_partition, kafka_offset,
			actor_id, actor_organization_id, ip_hash, correlation_id
		FROM audit_entries ORDER BY source, id`)
	require.NoError(t, err)

	defer rows.Close()

	var out []row

	for rows.Next() {
		var r row

		require.NoError(t, rows.Scan(
			&r.Source, &r.ID, &r.Type, &r.SpecVersion, &r.Subject, &r.Time, &r.DataContentType,
			&r.Headers, &r.Payload, &r.KafkaTopic, &r.KafkaPartition, &r.KafkaOffset,
			&r.ActorID, &r.ActorOrganizationID, &r.IPHash, &r.CorrelationID,
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
