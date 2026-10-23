package forwarder_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lcnascimento/go-kit/postgres/testdb"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/application/changes"
	changesrepo "github.com/lcnascimento/boilerplate/backend/services/iam/internal/repository/changes"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/repository/migrations"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/transport/worker/forwarder"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/transport/worker/translator"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/eventbustest"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"
)

const waitFor = 30 * time.Second

func TestComponentPipeline_DeliversTheIdentityChangesToTheEventBus(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	bus := eventbustest.New(t, events.Protos)
	for _, topic := range events.Topics {
		bus.CreateTopic(t, topic, 3)
		bus.CreateTopic(t, eventbus.DLQTopic(topic, bus.Group), 1)
	}

	schemaIDs, err := eventbus.LookupSchemaIDs(t.Context(), bus.Registry(t), events.Protos)
	require.NoError(t, err)

	pool := testdb.New(t, migrations.FS)
	startWorker(t, pool, schemaIDs, bus.Brokers)

	t.Run("the user lifecycle", func(t *testing.T) {
		received := subscribe(t, bus, events.UserTopic)
		userID := uuid.MustParse("3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b")

		signedUp := inTx(t, pool, `
			WITH new_user AS (
				INSERT INTO "user" (id, name, email, "emailVerified", "createdAt", "updatedAt")
				VALUES ($1, 'Ada Lovelace', 'ada@acme.test', false, now(), now())
				RETURNING id
			)
			INSERT INTO account (id, "accountId", "providerId", "userId", password, "createdAt", "updatedAt")
			SELECT gen_random_uuid(), id::text, 'credential', id, 'hash', now(), now() FROM new_user`, userID)
		passwordChanged := inRequest(t, pool, adaInAcme,
			`UPDATE account SET password = 'new-hash', "updatedAt" = now() WHERE "userId" = $1`, userID)
		deleted := inTx(t, pool, `DELETE FROM "user" WHERE id = $1`, userID)

		created, changed, gone := next(t, received), next(t, received), next(t, received)

		requireEvent(t, created, ada, signedUp, noRequest, &events.UserCreated{
			UserId:    "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
			CreatedAt: timestamppb.New(signedUp.at),
		})
		requireEvent(t, changed, ada, passwordChanged, adaInAcmeOrigin, &events.PasswordChanged{
			UserId:    "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
			ChangedAt: timestamppb.New(passwordChanged.at),
		})
		requireEvent(t, gone, ada, deleted, noRequest, &events.UserDeleted{
			UserId:    "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
			DeletedAt: timestamppb.New(deleted.at),
		})

		requireNoPendingChange(t, pool)
	})

	t.Run("the sessions and the failed sign-ins", func(t *testing.T) {
		received := subscribe(t, bus, events.SessionTopic)
		userID := uuid.MustParse("5b8e2f0a-4c1d-4e7f-9a3b-6c5d4e3f2a1b")
		sessionID := uuid.MustParse("8c4d2e1f-6a7b-4c3d-9e0f-1a2b3c4d5e6f")

		inTx(t, pool, `
			INSERT INTO "user" (id, name, email, "emailVerified", "createdAt", "updatedAt")
			VALUES ($1, 'Grace Hopper', 'grace@acme.test', false, now(), now())`, userID)

		signInFailed := inRequest(t, pool, anonymousFromGracesIP, `
			INSERT INTO identity_change (table_name, operation, fields, ip_hash)
			VALUES ('sign_in', 'FAILED', jsonb_build_object('reason', 'invalid_credentials'),
			        NULLIF(current_setting('app.ip_hash', true), ''))`)
		signedIn := inRequest(t, pool, anonymousFromGracesIP, `
			INSERT INTO session (id, "expiresAt", token, "updatedAt", "ipAddress", "userId", "activeOrganizationId")
			VALUES ($1, now() + interval '7 days', 'token', now(), '203.0.113.7', $2, '3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d')`,
			sessionID, userID)
		inRequest(t, pool, graceInAcme, `UPDATE session SET "updatedAt" = now() WHERE id = $1`, sessionID)
		signedOut := inRequest(t, pool, anonymousFromGracesIP, `DELETE FROM session WHERE id = $1`, sessionID)

		got := byType(t, received, 3)

		requireEvent(t, got["boilerplate.iam.v1.SignInFailed"], anonymous, signInFailed, origin{
			ipHash: "61370b3ab3601445f17092312e647e82fb0adbd5c0dd53592b53f2f4fbd59cc7",
		}, &events.SignInFailed{
			Reason:   events.SignInFailed_REASON_INVALID_CREDENTIALS,
			FailedAt: timestamppb.New(signInFailed.at),
		})
		requireEvent(t, got["boilerplate.iam.v1.SessionStarted"], gracesSession, signedIn, graceInAcmeOrigin, &events.SessionStarted{
			SessionId: "session_8c4d2e1f-6a7b-4c3d-9e0f-1a2b3c4d5e6f",
			UserId:    "user_5b8e2f0a-4c1d-4e7f-9a3b-6c5d4e3f2a1b",
			StartedAt: timestamppb.New(signedIn.at),
		})
		requireEvent(t, got["boilerplate.iam.v1.SessionEnded"], gracesSession, signedOut, graceInAcmeOrigin, &events.SessionEnded{
			SessionId: "session_8c4d2e1f-6a7b-4c3d-9e0f-1a2b3c4d5e6f",
			UserId:    "user_5b8e2f0a-4c1d-4e7f-9a3b-6c5d4e3f2a1b",
			EndedAt:   timestamppb.New(signedOut.at),
		})

		requireNoPendingChange(t, pool)
	})
}

type subject struct {
	topic string
	id    string
}

var (
	ada           = subject{topic: "iam.user.v1", id: "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b"}
	gracesSession = subject{topic: "iam.session.v1", id: "session_8c4d2e1f-6a7b-4c3d-9e0f-1a2b3c4d5e6f"}
	anonymous     = subject{topic: "iam.session.v1"}
)

type origin struct {
	actorID             string
	actorOrganizationID string
	ipHash              string
}

var (
	noRequest       = origin{}
	adaInAcmeOrigin = origin{
		actorID:             "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
		actorOrganizationID: "org_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d",
		ipHash:              "61370b3ab3601445f17092312e647e82fb0adbd5c0dd53592b53f2f4fbd59cc7",
	}
	graceInAcmeOrigin = origin{
		actorID:             "user_5b8e2f0a-4c1d-4e7f-9a3b-6c5d4e3f2a1b",
		actorOrganizationID: "org_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d",
		ipHash:              "61370b3ab3601445f17092312e647e82fb0adbd5c0dd53592b53f2f4fbd59cc7",
	}
)

func requireEvent(t *testing.T, got *eventbus.Event, about subject, tx transaction, from origin, want proto.Message) {
	t.Helper()

	require.NotNil(t, got, "no %s", want.ProtoReflect().Descriptor().FullName())
	require.True(t, proto.Equal(want, got.Message), "want %v, got %v", want, got.Message)
	require.Equal(t, about.topic, got.Topic)
	require.Equal(t, "iam", got.Source)
	require.Equal(t, string(want.ProtoReflect().Descriptor().FullName()), got.Type)
	require.Equal(t, about.id, got.Subject)
	require.Equal(t, tx.id, got.CorrelationID)
	require.True(t, tx.at.Equal(got.Time), "want %v, got %v", tx.at, got.Time)
	require.Equal(t, from, origin{
		actorID:             got.ActorID,
		actorOrganizationID: got.ActorOrganizationID,
		ipHash:              got.IPHash,
	})
}

type transaction struct {
	id string
	at time.Time
}

type connectionSettings struct {
	actorID             string
	actorOrganizationID string
	ipHash              string
}

var adaInAcme = connectionSettings{
	actorID:             "3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
	actorOrganizationID: "3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d",
	ipHash:              "61370b3ab3601445f17092312e647e82fb0adbd5c0dd53592b53f2f4fbd59cc7",
}

var graceInAcme = connectionSettings{
	actorID:             "5b8e2f0a-4c1d-4e7f-9a3b-6c5d4e3f2a1b",
	actorOrganizationID: "3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d",
	ipHash:              "61370b3ab3601445f17092312e647e82fb0adbd5c0dd53592b53f2f4fbd59cc7",
}

var anonymousFromGracesIP = connectionSettings{
	ipHash: "61370b3ab3601445f17092312e647e82fb0adbd5c0dd53592b53f2f4fbd59cc7",
}

func inTx(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) transaction {
	t.Helper()

	return inRequest(t, pool, connectionSettings{}, sql, args...)
}

func inRequest(t *testing.T, pool *pgxpool.Pool, settings connectionSettings, sql string, args ...any) transaction {
	t.Helper()

	var tx transaction

	require.NoError(t, pgx.BeginFunc(t.Context(), pool, func(conn pgx.Tx) error {
		if _, err := conn.Exec(t.Context(), `
			SELECT
				set_config('app.actor_id', $1, true),
				set_config('app.actor_organization_id', $2, true),
				set_config('app.ip_hash', $3, true)`,
			settings.actorID, settings.actorOrganizationID, settings.ipHash); err != nil {
			return err
		}

		if _, err := conn.Exec(t.Context(), sql, args...); err != nil {
			return err
		}

		return conn.QueryRow(t.Context(), `SELECT pg_current_xact_id()::text, now()`).Scan(&tx.id, &tx.at)
	}))

	return tx
}

func startWorker(t *testing.T, pool *pgxpool.Pool, schemaIDs eventbus.SchemaIDs, brokers []string) {
	t.Helper()

	worker := translator.New(changes.New(changesrepo.New(pool, schemaIDs)), translator.WithPollInterval(10*time.Millisecond))
	require.NoError(t, worker.Start(t.Context()))
	t.Cleanup(func() { require.NoError(t, worker.Stop(context.Background())) })

	f, err := forwarder.New(t.Context(), pool, brokers, func() { t.Errorf("forwarder stopped unexpectedly") })
	require.NoError(t, err)
	require.NoError(t, f.Start(t.Context()))
	require.NoError(t, f.Ready(t.Context()))
	t.Cleanup(func() { _ = f.Stop(context.Background()) })
}

type recorder chan *eventbus.Event

func (r recorder) Handle(_ context.Context, event *eventbus.Event) error {
	r <- event

	return nil
}

func subscribe(t *testing.T, bus *eventbustest.Env, topic string) recorder {
	t.Helper()

	received := make(recorder, 100)

	subscriber, err := eventbus.NewSubscriber(&eventbus.SubscriberConfig{
		Brokers: bus.Brokers,
		Group:   bus.Group,
		Topics:  []string{topic},
		Handler: received,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)

		_ = subscriber.Run(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		<-done

		_ = subscriber.Close()
	})

	select {
	case <-subscriber.Running():
	case <-time.After(waitFor):
		t.Fatal("events subscriber did not start")
	}

	return received
}

func next(t *testing.T, received recorder) *eventbus.Event {
	t.Helper()

	select {
	case event := <-received:
		return event
	case <-time.After(waitFor):
		t.Fatal("no event within the timeout")

		return nil
	}
}

func byType(t *testing.T, received recorder, count int) map[string]*eventbus.Event {
	t.Helper()

	got := make(map[string]*eventbus.Event, count)
	for range count {
		event := next(t, received)
		got[event.Type] = event
	}

	return got
}

func requireNoPendingChange(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	var pending int
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT count(*) FROM identity_change`).Scan(&pending))
	require.Zero(t, pending)
}
