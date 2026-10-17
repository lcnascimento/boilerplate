package outbox_test

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/outbox"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/testdb"
)

const (
	retention = 7 * 24 * time.Hour
	expired   = 8 * 24 * time.Hour
	recent    = 24 * time.Hour
)

func TestComponentRepository_PurgeKeepsPendingMessages(t *testing.T) {
	t.Parallel()

	pool, repo := newRepository(t)
	seed(t, pool, 2, expired)
	ack(t, pool)
	seed(t, pool, 3, expired)

	purged, err := repo.Purge(t.Context(), retention)

	require.NoError(t, err)
	require.Equal(t, int64(2), purged)
	require.Equal(t, 3, count(t, pool))
}

func TestComponentRepository_PurgeKeepsEverythingBeforeTheFirstAck(t *testing.T) {
	t.Parallel()

	pool, repo := newRepository(t)
	seed(t, pool, 3, expired)

	purged, err := repo.Purge(t.Context(), retention)

	require.NoError(t, err)
	require.Zero(t, purged)
	require.Equal(t, 3, count(t, pool))
}

func TestComponentRepository_PurgeKeepsRecentlyForwardedMessages(t *testing.T) {
	t.Parallel()

	pool, repo := newRepository(t)
	seed(t, pool, 3, recent)
	ack(t, pool)

	purged, err := repo.Purge(t.Context(), retention)

	require.NoError(t, err)
	require.Zero(t, purged)
	require.Equal(t, 3, count(t, pool))
}

func TestComponentRepository_PurgeDeletesExpiredForwardedMessagesAcrossBatches(t *testing.T) {
	t.Parallel()

	pool, repo := newRepository(t)
	seed(t, pool, 10_001, expired)
	seed(t, pool, 2, recent)
	ack(t, pool)

	purged, err := repo.Purge(t.Context(), retention)

	require.NoError(t, err)
	require.Equal(t, int64(10_001), purged)
	require.Equal(t, 2, count(t, pool))
}

func TestComponentRepository_OldestPendingAgeIsZeroWithoutMessages(t *testing.T) {
	t.Parallel()

	_, repo := newRepository(t)

	age, err := repo.OldestPendingAge(t.Context())

	require.NoError(t, err)
	require.Zero(t, age)
}

func TestComponentRepository_OldestPendingAgeIsZeroWhenEverythingWasForwarded(t *testing.T) {
	t.Parallel()

	pool, repo := newRepository(t)
	seed(t, pool, 3, time.Hour)
	ack(t, pool)

	age, err := repo.OldestPendingAge(t.Context())

	require.NoError(t, err)
	require.Zero(t, age)
}

func TestComponentRepository_OldestPendingAgeIgnoresForwardedMessages(t *testing.T) {
	t.Parallel()

	pool, repo := newRepository(t)
	seed(t, pool, 1, time.Hour)
	ack(t, pool)
	seed(t, pool, 1, 10*time.Minute)
	seed(t, pool, 1, time.Minute)

	age, err := repo.OldestPendingAge(t.Context())

	require.NoError(t, err)
	require.InDelta(t, (10 * time.Minute).Seconds(), age.Seconds(), 5)
}

func TestComponentRepository_OldestPendingAgeCountsEverythingBeforeTheFirstAck(t *testing.T) {
	t.Parallel()

	pool, repo := newRepository(t)
	seed(t, pool, 1, time.Hour)

	age, err := repo.OldestPendingAge(t.Context())

	require.NoError(t, err)
	require.InDelta(t, time.Hour.Seconds(), age.Seconds(), 5)
}

func TestComponentRepository_FollowsTheSubscriberCursor(t *testing.T) {
	t.Parallel()

	pool, repo := newRepository(t)
	publish(t, pool, "first")
	publish(t, pool, "second")

	subscriber, err := outbox.NewSubscriber(pool)
	require.NoError(t, err)
	t.Cleanup(func() { _ = subscriber.Close() })

	messages, err := subscriber.Subscribe(t.Context(), outbox.ForwarderTopic)
	require.NoError(t, err)

	(<-messages).Ack()

	require.Eventually(t, func() bool {
		purged, err := repo.Purge(t.Context(), 0)
		require.NoError(t, err)

		return purged == 1
	}, 10*time.Second, 50*time.Millisecond)

	age, err := repo.OldestPendingAge(t.Context())
	require.NoError(t, err)
	require.Positive(t, age)
	require.Equal(t, 1, count(t, pool))
}

func newRepository(t *testing.T) (*pgxpool.Pool, *outbox.Repository) {
	t.Helper()

	if testing.Short() {
		t.Skip("component test: needs Postgres")
	}

	pool := testdb.New(t)

	return pool, outbox.New(pool)
}

func seed(t *testing.T, pool *pgxpool.Pool, n int, age time.Duration) {
	t.Helper()

	_, err := pool.Exec(t.Context(), `
		INSERT INTO watermill_notes_outbox (uuid, created_at, payload, metadata, transaction_id)
		SELECT gen_random_uuid()::text, now() - make_interval(secs => $2), '{}', '{}', pg_current_xact_id()
		FROM generate_series(1, $1)`, n, age.Seconds())
	require.NoError(t, err)
}

func publish(t *testing.T, pool *pgxpool.Pool, uuid string) {
	t.Helper()

	tx, err := pool.Begin(t.Context())
	require.NoError(t, err)

	publisher, err := outbox.NewPublisher(tx)
	require.NoError(t, err)
	require.NoError(t, publisher.Publish("notes.note.v1", message.NewMessage(uuid, []byte("{}"))))
	require.NoError(t, tx.Commit(t.Context()))
}

func ack(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(t.Context(), `
		INSERT INTO watermill_offsets_notes_outbox (consumer_group, offset_acked, last_processed_transaction_id)
		SELECT $1, "offset", transaction_id
		FROM watermill_notes_outbox
		ORDER BY transaction_id DESC, "offset" DESC
		LIMIT 1
		ON CONFLICT (consumer_group) DO UPDATE
		SET offset_acked = excluded.offset_acked, last_processed_transaction_id = excluded.last_processed_transaction_id`,
		outbox.ConsumerGroup)
	require.NoError(t, err)
}

func count(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()

	var n int

	require.NoError(t, pool.QueryRow(t.Context(), `SELECT count(*) FROM watermill_notes_outbox`).Scan(&n))

	return n
}
