package notes_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/internal/testdb"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

var schemaIDs = events.SchemaIDs{
	"boilerplate.notes.v1.NoteCreated": 7,
	"boilerplate.notes.v1.NoteDeleted": 8,
}

func TestComponentRepository_CreateWritesNoteCreated(t *testing.T) {
	t.Parallel()

	repo, pool := newRepositoryWithPool(t)
	note := newNote("Lista de compras", time.Date(2026, 10, 10, 19, 30, 0, 123456000, time.UTC))
	note.Body = "Leite e pão"

	require.NoError(t, repo.Create(t.Context(), note))

	got := outboxEvents(t, pool)
	require.Len(t, got, 1)
	requireEnvelope(t, got[0], map[string]string{
		"ce_source":       "notes",
		"ce_type":         "boilerplate.notes.v1.NoteCreated",
		"ce_time":         "2026-10-10T19:30:00.123456Z",
		"ce_subject":      note.ID.String(),
		"ce_partitionkey": note.ID.String(),
		"ce_specversion":  "1.0",
		"content-type":    "application/protobuf",
	})
	require.Equal(t, 7, got[0].schemaID)
	requireProtoEqual(t, &events.NoteCreated{
		Id:        note.ID.String(),
		Title:     "Lista de compras",
		Body:      "Leite e pão",
		CreatedAt: timestamppb.New(time.Date(2026, 10, 10, 19, 30, 0, 123456000, time.UTC)),
	}, got[0].message)
}

func TestComponentRepository_DeleteWritesNoteDeletedAtClockTime(t *testing.T) {
	t.Parallel()

	deletedAt := time.Date(2026, 10, 11, 8, 15, 0, 0, time.UTC)
	repo, pool := newRepositoryWithPool(t, notes.WithClock(func() time.Time { return deletedAt }))
	note := newNote("Lista de compras", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))
	require.NoError(t, repo.Create(t.Context(), note))

	require.NoError(t, repo.Delete(t.Context(), note.ID))

	got := outboxEvents(t, pool)
	require.Len(t, got, 2)
	requireEnvelope(t, got[1], map[string]string{
		"ce_source":       "notes",
		"ce_type":         "boilerplate.notes.v1.NoteDeleted",
		"ce_time":         "2026-10-11T08:15:00Z",
		"ce_subject":      note.ID.String(),
		"ce_partitionkey": note.ID.String(),
		"ce_specversion":  "1.0",
		"content-type":    "application/protobuf",
	})
	require.Equal(t, 8, got[1].schemaID)
	requireProtoEqual(t, &events.NoteDeleted{Id: note.ID.String()}, got[1].message)
}

func TestComponentRepository_DeleteWithoutEffectWritesNothing(t *testing.T) {
	t.Parallel()

	repo, pool := newRepositoryWithPool(t)

	err := repo.Delete(t.Context(), domain.NewNoteID())

	require.ErrorIs(t, err, domain.ErrNoteNotFound)
	require.Empty(t, outboxEvents(t, pool))
}

func TestComponentRepository_FailedCreateWritesNothing(t *testing.T) {
	t.Parallel()

	repo, pool := newRepositoryWithPool(t)
	note := newNote("Lista de compras", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))
	require.NoError(t, repo.Create(t.Context(), note))

	err := repo.Create(t.Context(), note)

	require.ErrorIs(t, err, notes.ErrQuery)
	require.Len(t, outboxEvents(t, pool), 1)
}

func TestComponentRepository_ConcurrentDeletesWriteOneNoteDeleted(t *testing.T) {
	t.Parallel()

	repo, pool := newRepositoryWithPool(t)
	note := newNote("Lista de compras", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))
	require.NoError(t, repo.Create(t.Context(), note))

	const deletes = 8

	errs := make([]error, deletes)

	var wg sync.WaitGroup

	for i := range deletes {
		wg.Go(func() { errs[i] = repo.Delete(t.Context(), note.ID) })
	}

	wg.Wait()

	succeeded := 0

	for _, err := range errs {
		if err == nil {
			succeeded++

			continue
		}

		require.ErrorIs(t, err, domain.ErrNoteNotFound)
	}

	require.Equal(t, 1, succeeded)
	require.Equal(t, []string{"boilerplate.notes.v1.NoteCreated", "boilerplate.notes.v1.NoteDeleted"}, eventTypes(outboxEvents(t, pool)))
}

func TestComponentRepository_OutboxFailureLeavesNoNote(t *testing.T) {
	t.Parallel()

	repo, pool := newRepositoryWithPool(t)
	_, err := pool.Exec(t.Context(), `DROP TABLE watermill_notes_outbox`)
	require.NoError(t, err)

	note := newNote("Lista de compras", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))

	err = repo.Create(t.Context(), note)

	require.ErrorIs(t, err, notes.ErrEmit)

	_, err = repo.Get(t.Context(), note.ID)
	require.ErrorIs(t, err, domain.ErrNoteNotFound)
}

func TestComponentRepository_UnregisteredSchemaLeavesNoNote(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("component test: needs Postgres")
	}

	repo := notes.New(testdb.New(t), events.SchemaIDs{})
	note := newNote("Lista de compras", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))

	err := repo.Create(t.Context(), note)

	require.ErrorIs(t, err, events.ErrUnknownSchema)

	_, err = repo.Get(t.Context(), note.ID)
	require.ErrorIs(t, err, domain.ErrNoteNotFound)
}

type outboxEvent struct {
	destination string
	uuid        string
	headers     map[string]string
	schemaID    int
	message     proto.Message
}

// outboxEvents reads the outbox in write order, unwrapping the forwarder's JSON envelope.
func outboxEvents(t *testing.T, pool *pgxpool.Pool) []outboxEvent {
	t.Helper()

	rows, err := pool.Query(t.Context(), `SELECT payload FROM watermill_notes_outbox ORDER BY transaction_id, "offset"`)
	require.NoError(t, err)

	defer rows.Close()

	var out []outboxEvent

	for rows.Next() {
		var raw []byte
		require.NoError(t, rows.Scan(&raw))

		var envelope struct {
			DestinationTopic string            `json:"destination_topic"`
			UUID             string            `json:"uuid"`
			Payload          []byte            `json:"payload"`
			Metadata         map[string]string `json:"metadata"`
		}
		require.NoError(t, json.Unmarshal(raw, &envelope))

		schemaID, message, err := events.Decode(envelope.Metadata["ce_type"], envelope.Payload)
		require.NoError(t, err)

		out = append(out, outboxEvent{
			destination: envelope.DestinationTopic,
			uuid:        envelope.UUID,
			headers:     envelope.Metadata,
			schemaID:    schemaID,
			message:     message,
		})
	}

	require.NoError(t, rows.Err())

	return out
}

// requireEnvelope checks the event goes to notes.note.v1 with want as its CloudEvents headers
// and a UUIDv7 ce_id that is also the Watermill message UUID.
func requireEnvelope(t *testing.T, got outboxEvent, want map[string]string) {
	t.Helper()

	require.Equal(t, "notes.note.v1", got.destination)

	id, err := uuid.Parse(got.headers["ce_id"])
	require.NoError(t, err)
	require.Equal(t, uuid.Version(7), id.Version())
	require.Equal(t, got.uuid, got.headers["ce_id"])

	headers := map[string]string{}

	for key := range want {
		headers[key] = got.headers[key]
	}

	require.Equal(t, want, headers)
}

func eventTypes(list []outboxEvent) []string {
	out := make([]string, 0, len(list))
	for _, event := range list {
		out = append(out, event.headers["ce_type"])
	}

	return out
}

func requireProtoEqual(t *testing.T, want, got proto.Message) {
	t.Helper()

	require.True(t, proto.Equal(want, got), "want %v, got %v", want, got)
}
