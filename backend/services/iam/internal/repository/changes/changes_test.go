package changes_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lcnascimento/go-kit/postgres/testdb"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/idomain"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/repository/changes"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/repository/migrations"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"
)

const batch = 100

var (
	userCreatedType     = string((&events.UserCreated{}).ProtoReflect().Descriptor().FullName())
	userDeletedType     = string((&events.UserDeleted{}).ProtoReflect().Descriptor().FullName())
	passwordChangedType = string((&events.PasswordChanged{}).ProtoReflect().Descriptor().FullName())
)

var schemaIDs = eventbus.SchemaIDs{
	userCreatedType:     11,
	userDeletedType:     12,
	passwordChangedType: 13,
}

var ada = uuid.MustParse("3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b")

func TestComponentRepository_SignUpWritesOneUserCreated(t *testing.T) {
	t.Parallel()

	repo, pool := newRepository(t)
	tx := signUp(t, pool, ada)

	translated, err := repo.Translate(t.Context(), batch)

	require.NoError(t, err)
	require.Len(t, translated, 1)
	require.Equal(t, "boilerplate.iam.v1.UserCreated", translated[0].Type())

	got := outboxEvents(t, pool)
	require.Len(t, got, 1)
	requireEnvelope(t, got[0], map[string]string{
		"ce_source":        "iam",
		"ce_type":          "boilerplate.iam.v1.UserCreated",
		"ce_time":          tx.at.UTC().Format(time.RFC3339Nano),
		"ce_subject":       "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
		"ce_partitionkey":  "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
		"ce_specversion":   "1.0",
		"content-type":     "application/protobuf",
		"ce_correlationid": tx.id,
	})
	require.Equal(t, 11, got[0].schemaID)
	requireProtoEqual(t, &events.UserCreated{
		UserId:    "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
		CreatedAt: timestamppb.New(tx.at),
	}, got[0].message)
}

func TestComponentRepository_PasswordUpdateWritesPasswordChanged(t *testing.T) {
	t.Parallel()

	repo, pool := newRepository(t)
	signUp(t, pool, ada)
	tx := inTx(t, pool, `UPDATE account SET password = 'new-hash', "updatedAt" = now() WHERE "userId" = $1`, ada)

	_, err := repo.Translate(t.Context(), batch)
	require.NoError(t, err)

	got := outboxEvents(t, pool)
	require.Equal(t, []string{userCreatedType, passwordChangedType}, eventTypes(got))
	require.Equal(t, tx.id, got[1].headers["ce_correlationid"])
	requireProtoEqual(t, &events.PasswordChanged{
		UserId:    "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
		ChangedAt: timestamppb.New(tx.at),
	}, got[1].message)
}

func TestComponentRepository_AccountUpdateWithoutANewPasswordWritesNothing(t *testing.T) {
	t.Parallel()

	repo, pool := newRepository(t)
	signUp(t, pool, ada)
	inTx(t, pool, `UPDATE account SET password = password, "updatedAt" = now() WHERE "userId" = $1`, ada)
	inTx(t, pool, `UPDATE "user" SET name = 'Ada King', "updatedAt" = now() WHERE id = $1`, ada)

	_, err := repo.Translate(t.Context(), batch)
	require.NoError(t, err)

	require.Equal(t, []string{userCreatedType}, eventTypes(outboxEvents(t, pool)))
}

func TestComponentRepository_UserDeletionWritesOneUserDeleted(t *testing.T) {
	t.Parallel()

	repo, pool := newRepository(t)
	signUp(t, pool, ada)
	tx := inTx(t, pool, `DELETE FROM "user" WHERE id = $1`, ada)

	_, err := repo.Translate(t.Context(), batch)
	require.NoError(t, err)

	got := outboxEvents(t, pool)
	require.Equal(t, []string{userCreatedType, userDeletedType}, eventTypes(got))
	require.Equal(t, tx.id, got[1].headers["ce_correlationid"])
	requireProtoEqual(t, &events.UserDeleted{
		UserId:    "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
		DeletedAt: timestamppb.New(tx.at),
	}, got[1].message)
}

func TestComponentRepository_EventsOfOneTransactionShareTheCorrelationID(t *testing.T) {
	t.Parallel()

	repo, pool := newRepository(t)
	grace := uuid.MustParse("9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f")

	signUp(t, pool, ada)
	signUp(t, pool, grace)
	tx := inTx(t, pool, `DELETE FROM "user" WHERE id = ANY($1)`, []uuid.UUID{ada, grace})

	_, err := repo.Translate(t.Context(), batch)
	require.NoError(t, err)

	got := outboxEvents(t, pool)
	require.Equal(t, []string{userCreatedType, userCreatedType, userDeletedType, userDeletedType}, eventTypes(got))
	require.NotEqual(t, got[0].headers["ce_correlationid"], got[1].headers["ce_correlationid"])
	require.Equal(t, tx.id, got[2].headers["ce_correlationid"])
	require.Equal(t, tx.id, got[3].headers["ce_correlationid"])
}

func TestComponentRepository_EventsCarryNoEmailOrName(t *testing.T) {
	t.Parallel()

	repo, pool := newRepository(t)
	signUp(t, pool, ada)
	inTx(t, pool, `UPDATE account SET password = 'new-hash' WHERE "userId" = $1`, ada)
	inTx(t, pool, `DELETE FROM "user" WHERE id = $1`, ada)

	_, err := repo.Translate(t.Context(), batch)
	require.NoError(t, err)

	var outbox string
	require.NoError(t, pool.QueryRow(t.Context(),
		`SELECT string_agg(payload::text || metadata::text, '') FROM watermill_iam_outbox`).Scan(&outbox))
	require.NotContains(t, outbox, "ada@acme.test")
	require.NotContains(t, outbox, "Ada Lovelace")
	require.NotContains(t, outbox, "new-hash")
}

func TestComponentRepository_TranslatedChangesAreGone(t *testing.T) {
	t.Parallel()

	repo, pool := newRepository(t)
	signUp(t, pool, ada)

	_, err := repo.Translate(t.Context(), batch)
	require.NoError(t, err)

	require.Zero(t, pendingChanges(t, pool))

	translated, err := repo.Translate(t.Context(), batch)
	require.NoError(t, err)
	require.Empty(t, translated)
	require.Len(t, outboxEvents(t, pool), 1)
}

func TestComponentRepository_TranslatesAtMostTheLimit(t *testing.T) {
	t.Parallel()

	repo, pool := newRepository(t)

	for range 3 {
		signUp(t, pool, uuid.New())
	}

	translated, err := repo.Translate(t.Context(), 2)

	require.NoError(t, err)
	require.Len(t, translated, 2)
	require.Equal(t, 1, pendingChanges(t, pool))
	require.Len(t, outboxEvents(t, pool), 2)
}

func TestComponentRepository_UntranslatableChangeHoldsBackTheBatch(t *testing.T) {
	t.Parallel()

	repo, pool := newRepository(t)
	signUp(t, pool, ada)
	inTx(t, pool, `INSERT INTO identity_change (table_name, operation, fields) VALUES ('verification', 'INSERT', '{}')`)

	_, err := repo.Translate(t.Context(), batch)

	require.ErrorIs(t, err, idomain.ErrUntranslatableChange)
	require.Equal(t, 2, pendingChanges(t, pool))
	require.Empty(t, outboxEvents(t, pool))
}

func TestComponentRepository_OldestPendingAgeIsZeroWithoutPendingChanges(t *testing.T) {
	t.Parallel()

	repo, _ := newRepository(t)

	age, err := repo.OldestPendingAge(t.Context())

	require.NoError(t, err)
	require.Zero(t, age)
}

func TestComponentRepository_OldestPendingAgeIsTheAgeOfTheOldestChange(t *testing.T) {
	t.Parallel()

	repo, pool := newRepository(t)
	inTx(t, pool, `INSERT INTO identity_change (occurred_at, table_name, operation, fields)
		VALUES (now() - interval '10 minutes', 'user', 'INSERT', '{}'), (now() - interval '1 minute', 'user', 'INSERT', '{}')`)

	age, err := repo.OldestPendingAge(t.Context())

	require.NoError(t, err)
	require.InDelta(t, (10 * time.Minute).Seconds(), age.Seconds(), 5)
}

func TestComponentRepository_ConcurrentStepsTranslateEachChangeOnce(t *testing.T) {
	t.Parallel()

	repo, pool := newRepository(t)

	const users = 20

	for range users {
		signUp(t, pool, uuid.New())
	}

	const steps = 4

	var wg sync.WaitGroup

	errs := make([]error, steps)

	for i := range steps {
		wg.Go(func() {
			for {
				translated, err := repo.Translate(t.Context(), 3)
				if err != nil || len(translated) == 0 {
					errs[i] = err

					return
				}
			}
		})
	}

	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}

	subjects := map[string]int{}
	for _, event := range outboxEvents(t, pool) {
		subjects[event.headers["ce_subject"]]++
	}

	require.Len(t, subjects, users)

	for subject, count := range subjects {
		require.Equal(t, 1, count, subject)
	}
}

type transaction struct {
	id string
	at time.Time
}

func signUp(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) transaction {
	t.Helper()

	return inTx(t, pool, `
		WITH new_user AS (
			INSERT INTO "user" (id, name, email, "emailVerified", "createdAt", "updatedAt")
			VALUES ($1, 'Ada Lovelace', $2, false, now(), now())
			RETURNING id
		)
		INSERT INTO account (id, "accountId", "providerId", "userId", password, "createdAt", "updatedAt")
		SELECT gen_random_uuid(), id::text, 'credential', id, 'hash', now(), now() FROM new_user`,
		userID, emailOf(userID))
}

func emailOf(userID uuid.UUID) string {
	if userID == ada {
		return "ada@acme.test"
	}

	return userID.String() + "@acme.test"
}

func inTx(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) transaction {
	t.Helper()

	var tx transaction

	require.NoError(t, pgx.BeginFunc(t.Context(), pool, func(conn pgx.Tx) error {
		if _, err := conn.Exec(t.Context(), sql, args...); err != nil {
			return err
		}

		return conn.QueryRow(t.Context(), `SELECT pg_current_xact_id()::text, now()`).Scan(&tx.id, &tx.at)
	}))

	return tx
}

func pendingChanges(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()

	var count int
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT count(*) FROM identity_change`).Scan(&count))

	return count
}

type outboxEvent struct {
	destination string
	uuid        string
	headers     map[string]string
	schemaID    int
	message     proto.Message
}

func outboxEvents(t *testing.T, pool *pgxpool.Pool) []outboxEvent {
	t.Helper()

	rows, err := pool.Query(t.Context(), `SELECT payload FROM watermill_iam_outbox ORDER BY transaction_id, "offset"`)
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

		schemaID, message, err := eventbus.Decode(envelope.Metadata["ce_type"], envelope.Payload)
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

func requireEnvelope(t *testing.T, got outboxEvent, want map[string]string) {
	t.Helper()

	require.Equal(t, "iam.user.v1", got.destination)

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

func newRepository(t *testing.T) (*changes.Repository, *pgxpool.Pool) {
	t.Helper()

	if testing.Short() {
		t.Skip("component test")
	}

	pool := testdb.New(t, migrations.FS)

	return changes.New(pool, schemaIDs), pool
}
