package forwarder_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lcnascimento/go-kit/postgres/testdb"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/repository/migrations"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"
)

const (
	acmeID  = "org_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d"
	adaID   = "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b"
	graceID = "user_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f"

	adaMemberID   = "member_6d0e3b2a-4c5f-4e8d-9a1b-2c3d4e5f6a7b"
	graceMemberID = "member_1e2d3c4b-5a69-4788-a96b-5c4d3e2f1a0b"

	graceInvitationID = "invitation_8c4f2e1d-0b3a-4d5c-8e6f-7a8b9c0d1e2f"
	linusInvitationID = "invitation_2b3c4d5e-6f70-4182-93a4-b5c6d7e8f901"
	alanInvitationID  = "invitation_4d5e6f70-8192-4a3b-8c5d-6e7f8091a2b3"

	ipHash = "61370b3ab3601445f17092312e647e82fb0adbd5c0dd53592b53f2f4fbd59cc7"
)

var (
	acme = aggregate{topic: "iam.organization.v1", subject: acmeID}

	adaAlone = connectionSettings{
		actorID: "3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
		ipHash:  ipHash,
	}
	graceAlone = connectionSettings{
		actorID: "9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f",
		ipHash:  ipHash,
	}
	graceInAcme = connectionSettings{
		actorID:             "9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f",
		actorOrganizationID: "3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d",
		ipHash:              ipHash,
	}
	linusAlone = connectionSettings{
		actorID: "5e6f7081-92a3-4b4c-9d5e-6f708192a3b4",
		ipHash:  ipHash,
	}

	fromAda         = origin{actorID: adaID, ipHash: ipHash}
	fromAdaInAcme   = origin{actorID: adaID, actorOrganizationID: acmeID, ipHash: ipHash}
	fromGrace       = origin{actorID: graceID, ipHash: ipHash}
	fromGraceInAcme = origin{actorID: graceID, actorOrganizationID: acmeID, ipHash: ipHash}
	fromLinus       = origin{actorID: "user_5e6f7081-92a3-4b4c-9d5e-6f708192a3b4", ipHash: ipHash}
)

func TestComponentPipeline_DeliversTheOrganizationLifecycleToTheEventBus(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	bus := newBus(t)

	schemaIDs, err := eventbus.LookupSchemaIDs(t.Context(), bus.Registry(t), events.Protos)
	require.NoError(t, err)

	pool := testdb.New(t, migrations.FS)

	inTx(t, pool, `
		INSERT INTO "user" (id, name, email, "emailVerified", "createdAt", "updatedAt") VALUES
			('3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b', 'Ada Lovelace', 'ada@acme.test', false, now(), now()),
			('9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f', 'Grace Hopper', 'grace@acme.test', false, now(), now()),
			('5e6f7081-92a3-4b4c-9d5e-6f708192a3b4', 'Linus Torvalds', 'linus@acme.test', false, now(), now());
		DELETE FROM identity_change`)

	created := inRequest(t, pool, adaAlone, `
		INSERT INTO organization (id, name, slug, "createdAt")
		VALUES ('3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d', 'Acme', 'acme', now());
		INSERT INTO member (id, "organizationId", "userId", role, "createdAt")
		VALUES ('6d0e3b2a-4c5f-4e8d-9a1b-2c3d4e5f6a7b', '3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d',
			'3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b', 'owner', now())`)
	inRequest(t, pool, adaInAcme,
		`UPDATE organization SET slug = 'acme-corp' WHERE id = '3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d'`)
	renamed := inRequest(t, pool, adaInAcme,
		`UPDATE organization SET name = 'Acme Corp' WHERE id = '3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d'`)
	invitedGrace := inRequest(t, pool, adaInAcme, `
		INSERT INTO invitation (id, "organizationId", email, role, status, "expiresAt", "inviterId")
		VALUES ('8c4f2e1d-0b3a-4d5c-8e6f-7a8b9c0d1e2f', '3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d',
			'grace@acme.test', 'viewer', 'pending', now() + interval '7 days', '3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b')`)
	accepted := inRequest(t, pool, graceAlone,
		`UPDATE invitation SET status = 'accepted' WHERE id = '8c4f2e1d-0b3a-4d5c-8e6f-7a8b9c0d1e2f'`)
	joined := inRequest(t, pool, graceAlone, `
		INSERT INTO member (id, "organizationId", "userId", role, "createdAt")
		VALUES ('1e2d3c4b-5a69-4788-a96b-5c4d3e2f1a0b', '3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d',
			'9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f', 'viewer', now())`)
	promoted := inRequest(t, pool, adaInAcme,
		`UPDATE member SET role = 'admin' WHERE id = '1e2d3c4b-5a69-4788-a96b-5c4d3e2f1a0b'`)
	transferred := inRequest(t, pool, adaInAcme, `
		UPDATE member SET role = 'owner' WHERE id = '1e2d3c4b-5a69-4788-a96b-5c4d3e2f1a0b';
		UPDATE member SET role = 'admin' WHERE id = '6d0e3b2a-4c5f-4e8d-9a1b-2c3d4e5f6a7b'`)
	invitedMore := inRequest(t, pool, graceInAcme, `
		INSERT INTO invitation (id, "organizationId", email, role, status, "expiresAt", "inviterId") VALUES
			('2b3c4d5e-6f70-4182-93a4-b5c6d7e8f901', '3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d',
				'linus@acme.test', 'viewer', 'pending', now() + interval '7 days', '9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f'),
			('4d5e6f70-8192-4a3b-8c5d-6e7f8091a2b3', '3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d',
				'alan@acme.test', 'admin', 'pending', now() + interval '7 days', '9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f')`)
	declined := inRequest(t, pool, linusAlone,
		`UPDATE invitation SET status = 'rejected' WHERE id = '2b3c4d5e-6f70-4182-93a4-b5c6d7e8f901'`)
	canceled := inRequest(t, pool, graceInAcme,
		`UPDATE invitation SET status = 'canceled' WHERE id = '4d5e6f70-8192-4a3b-8c5d-6e7f8091a2b3'`)
	left := inRequest(t, pool, adaInAcme,
		`DELETE FROM member WHERE id = '6d0e3b2a-4c5f-4e8d-9a1b-2c3d4e5f6a7b'`)
	deleted := inRequest(t, pool, graceInAcme,
		`DELETE FROM organization WHERE id = '3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d'`)

	const organizationEvents, invitationEvents = 10, 6

	var pending int
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT count(*) FROM identity_change`).Scan(&pending))
	require.Equal(t, organizationEvents+invitationEvents, pending)

	received := subscribe(t, bus, events.OrganizationTopic, events.InvitationTopic)
	startWorker(t, pool, schemaIDs, bus.Brokers)

	var ofAcme []*eventbus.Event

	ofInvitation := map[string][]*eventbus.Event{}

	for range organizationEvents + invitationEvents {
		event := next(t, received)
		if event.Topic == events.OrganizationTopic {
			ofAcme = append(ofAcme, event)
		} else {
			ofInvitation[event.Subject] = append(ofInvitation[event.Subject], event)
		}
	}

	require.Len(t, ofAcme, organizationEvents)
	requireEvent(t, ofAcme[0], acme, created, fromAda, &events.OrganizationCreated{
		OrganizationId: acmeID,
		CreatedAt:      timestamppb.New(created.at),
	})
	requireEvent(t, ofAcme[1], acme, created, fromAda, &events.MemberAdded{
		MemberId:       adaMemberID,
		OrganizationId: acmeID,
		UserId:         adaID,
		Role:           "owner",
		AddedAt:        timestamppb.New(created.at),
	})
	requireEvent(t, ofAcme[2], acme, renamed, fromAdaInAcme, &events.OrganizationRenamed{
		OrganizationId: acmeID,
		RenamedAt:      timestamppb.New(renamed.at),
	})
	requireEvent(t, ofAcme[3], acme, joined, fromGrace, &events.MemberAdded{
		MemberId:       graceMemberID,
		OrganizationId: acmeID,
		UserId:         graceID,
		Role:           "viewer",
		AddedAt:        timestamppb.New(joined.at),
	})
	requireEvent(t, ofAcme[4], acme, promoted, fromAdaInAcme, &events.MemberRoleChanged{
		MemberId:       graceMemberID,
		OrganizationId: acmeID,
		UserId:         graceID,
		Role:           "admin",
		ChangedAt:      timestamppb.New(promoted.at),
	})
	requireEvent(t, ofAcme[5], acme, transferred, fromAdaInAcme, &events.MemberRoleChanged{
		MemberId:       graceMemberID,
		OrganizationId: acmeID,
		UserId:         graceID,
		Role:           "owner",
		ChangedAt:      timestamppb.New(transferred.at),
	})
	requireEvent(t, ofAcme[6], acme, transferred, fromAdaInAcme, &events.MemberRoleChanged{
		MemberId:       adaMemberID,
		OrganizationId: acmeID,
		UserId:         adaID,
		Role:           "admin",
		ChangedAt:      timestamppb.New(transferred.at),
	})
	requireEvent(t, ofAcme[7], acme, left, fromAdaInAcme, &events.MemberRemoved{
		MemberId:       adaMemberID,
		OrganizationId: acmeID,
		UserId:         adaID,
		RemovedAt:      timestamppb.New(left.at),
	})
	requireEvent(t, ofAcme[8], acme, deleted, fromGraceInAcme, &events.OrganizationDeleted{
		OrganizationId: acmeID,
		DeletedAt:      timestamppb.New(deleted.at),
	})
	requireEvent(t, ofAcme[9], acme, deleted, fromGraceInAcme, &events.MemberRemoved{
		MemberId:       graceMemberID,
		OrganizationId: acmeID,
		UserId:         graceID,
		RemovedAt:      timestamppb.New(deleted.at),
	})

	grace := aggregate{topic: "iam.invitation.v1", subject: graceInvitationID}
	require.Len(t, ofInvitation[graceInvitationID], 2)
	requireEvent(t, ofInvitation[graceInvitationID][0], grace, invitedGrace, fromAdaInAcme, &events.InvitationCreated{
		InvitationId:   graceInvitationID,
		OrganizationId: acmeID,
		InviterId:      adaID,
		Role:           "viewer",
		CreatedAt:      timestamppb.New(invitedGrace.at),
	})
	requireEvent(t, ofInvitation[graceInvitationID][1], grace, accepted, fromGrace, &events.InvitationAccepted{
		InvitationId:   graceInvitationID,
		OrganizationId: acmeID,
		AcceptedAt:     timestamppb.New(accepted.at),
	})

	linus := aggregate{topic: "iam.invitation.v1", subject: linusInvitationID}
	require.Len(t, ofInvitation[linusInvitationID], 2)
	requireEvent(t, ofInvitation[linusInvitationID][0], linus, invitedMore, fromGraceInAcme, &events.InvitationCreated{
		InvitationId:   linusInvitationID,
		OrganizationId: acmeID,
		InviterId:      graceID,
		Role:           "viewer",
		CreatedAt:      timestamppb.New(invitedMore.at),
	})
	requireEvent(t, ofInvitation[linusInvitationID][1], linus, declined, fromLinus, &events.InvitationDeclined{
		InvitationId:   linusInvitationID,
		OrganizationId: acmeID,
		DeclinedAt:     timestamppb.New(declined.at),
	})

	alan := aggregate{topic: "iam.invitation.v1", subject: alanInvitationID}
	require.Len(t, ofInvitation[alanInvitationID], 2)
	requireEvent(t, ofInvitation[alanInvitationID][0], alan, invitedMore, fromGraceInAcme, &events.InvitationCreated{
		InvitationId:   alanInvitationID,
		OrganizationId: acmeID,
		InviterId:      graceID,
		Role:           "admin",
		CreatedAt:      timestamppb.New(invitedMore.at),
	})
	requireEvent(t, ofInvitation[alanInvitationID][1], alan, canceled, fromGraceInAcme, &events.InvitationCanceled{
		InvitationId:   alanInvitationID,
		OrganizationId: acmeID,
		CanceledAt:     timestamppb.New(canceled.at),
	})

	require.NoError(t, pool.QueryRow(t.Context(), `SELECT count(*) FROM identity_change`).Scan(&pending))
	require.Zero(t, pending)
}
