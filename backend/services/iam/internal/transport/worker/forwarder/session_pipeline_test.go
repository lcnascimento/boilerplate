package forwarder_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lcnascimento/go-kit/postgres/testdb"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/repository/migrations"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"
)

const (
	hedyID        = "user_5b8e2f0a-4c1d-4e7f-9a3b-6c5d4e3f2a1b"
	hedySessionID = "session_8c4d2e1f-6a7b-4c3d-9e0f-1a2b3c4d5e6f"
)

var (
	hedysSession = aggregate{topic: "iam.session.v1", subject: hedySessionID}
	signIns      = aggregate{topic: "iam.session.v1"}

	hedyInAcme = connectionSettings{
		actorID:             "5b8e2f0a-4c1d-4e7f-9a3b-6c5d4e3f2a1b",
		actorOrganizationID: "3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d",
		ipHash:              ipHash,
	}
	anonymousFromHedysIP = connectionSettings{ipHash: ipHash}

	fromHedyInAcme = origin{actorID: hedyID, actorOrganizationID: acmeID, ipHash: ipHash}
	fromHedysIP    = origin{ipHash: ipHash}
)

func TestComponentPipeline_DeliversTheSessionsAndTheFailedSignInsToTheEventBus(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	bus := newBus(t)

	schemaIDs, err := eventbus.LookupSchemaIDs(t.Context(), bus.Registry(t), events.Protos)
	require.NoError(t, err)

	pool := testdb.New(t, migrations.FS)
	userID := uuid.MustParse("5b8e2f0a-4c1d-4e7f-9a3b-6c5d4e3f2a1b")
	sessionID := uuid.MustParse("8c4d2e1f-6a7b-4c3d-9e0f-1a2b3c4d5e6f")

	inTx(t, pool, `
		INSERT INTO "user" (id, name, email, "emailVerified", "createdAt", "updatedAt")
		VALUES ($1, 'Hedy Lamarr', 'hedy@acme.test', false, now(), now())`, userID)

	signInFailed := inRequest(t, pool, anonymousFromHedysIP, `
		INSERT INTO identity_change (table_name, operation, fields, ip_hash)
		VALUES ('sign_in', 'FAILED', jsonb_build_object('reason', 'invalid_credentials'),
		        NULLIF(current_setting('app.ip_hash', true), ''))`)
	signedIn := inRequest(t, pool, anonymousFromHedysIP, `
		INSERT INTO session (id, "expiresAt", token, "updatedAt", "ipAddress", "userId", "activeOrganizationId")
		VALUES ($1, now() + interval '7 days', 'token', now(), '203.0.113.7', $2, '3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d')`,
		sessionID, userID)
	inRequest(t, pool, hedyInAcme, `UPDATE session SET "updatedAt" = now() WHERE id = $1`, sessionID)
	signedOut := inRequest(t, pool, anonymousFromHedysIP, `DELETE FROM session WHERE id = $1`, sessionID)

	received := subscribe(t, bus, events.SessionTopic)
	startWorker(t, pool, schemaIDs, bus.Brokers)

	got := map[string]*eventbus.Event{}

	for range 3 {
		event := next(t, received)
		got[event.Type] = event
	}

	require.Len(t, got, 3)
	requireEvent(t, got["boilerplate.iam.v1.SignInFailed"], signIns, signInFailed, fromHedysIP, &events.SignInFailed{
		Reason:   events.SignInFailed_REASON_INVALID_CREDENTIALS,
		FailedAt: timestamppb.New(signInFailed.at),
	})
	requireEvent(t, got["boilerplate.iam.v1.SessionStarted"], hedysSession, signedIn, fromHedyInAcme, &events.SessionStarted{
		SessionId: hedySessionID,
		UserId:    hedyID,
		StartedAt: timestamppb.New(signedIn.at),
	})
	requireEvent(t, got["boilerplate.iam.v1.SessionEnded"], hedysSession, signedOut, fromHedyInAcme, &events.SessionEnded{
		SessionId: hedySessionID,
		UserId:    hedyID,
		EndedAt:   timestamppb.New(signedOut.at),
	})

	var pending int
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT count(*) FROM identity_change`).Scan(&pending))
	require.Zero(t, pending)
}
