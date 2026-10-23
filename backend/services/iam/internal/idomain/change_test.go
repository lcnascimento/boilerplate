package idomain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/idomain"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"
)

var occurredAt = time.Date(2026, 10, 22, 18, 30, 0, 123456000, time.UTC)

func TestChange_EventTranslatesEachUserChange(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		change idomain.Change
		want   proto.Message
	}{
		"inserted user": {
			change: idomain.Change{Table: "user", Operation: "INSERT", Fields: map[string]string{
				"id": "3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
			}},
			want: &events.UserCreated{
				UserId:    "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
				CreatedAt: timestamppb.New(occurredAt),
			},
		},
		"deleted user": {
			change: idomain.Change{Table: "user", Operation: "DELETE", Fields: map[string]string{
				"id": "3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
			}},
			want: &events.UserDeleted{
				UserId:    "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
				DeletedAt: timestamppb.New(occurredAt),
			},
		},
		"updated account password": {
			change: idomain.Change{Table: "account", Operation: "UPDATE", Fields: map[string]string{
				"userId": "3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
			}},
			want: &events.PasswordChanged{
				UserId:    "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
				ChangedAt: timestamppb.New(occurredAt),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.change.TransactionID = "7311"
			tc.change.OccurredAt = occurredAt

			got, err := tc.change.Event()

			require.NoError(t, err)
			require.True(t, proto.Equal(tc.want, got.Message), "want %v, got %v", tc.want, got.Message)
			require.Equal(t, "iam.user.v1", got.Topic)
			require.Equal(t, "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b", got.Subject)
			require.Equal(t, occurredAt, got.Time)
			require.Equal(t, "7311", got.CorrelationID)
		})
	}
}

func TestChange_EventCarriesTheActorAndTheIPHash(t *testing.T) {
	t.Parallel()

	change := idomain.Change{
		Table:     "account",
		Operation: "UPDATE",
		Fields:    map[string]string{"userId": "3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b"},
		ActorID:   uuid.NullUUID{UUID: uuid.MustParse("9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f"), Valid: true},
		ActorOrganizationID: uuid.NullUUID{
			UUID:  uuid.MustParse("3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d"),
			Valid: true,
		},
		IPHash: "5d1f0c2a",
	}

	got, err := change.Event()

	require.NoError(t, err)
	require.Equal(t, "user_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f", got.ActorID)
	require.Equal(t, "org_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d", got.ActorOrganizationID)
	require.Equal(t, "5d1f0c2a", got.IPHash)
}

func TestChange_EventWithoutAnActorHasEmptyActorAndIPHash(t *testing.T) {
	t.Parallel()

	change := idomain.Change{Table: "user", Operation: "INSERT", Fields: map[string]string{
		"id": "3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
	}}

	got, err := change.Event()

	require.NoError(t, err)
	require.Empty(t, got.ActorID)
	require.Empty(t, got.ActorOrganizationID)
	require.Empty(t, got.IPHash)
}

func TestChange_EventRefusesAChangeWithoutAnIntegrationEvent(t *testing.T) {
	t.Parallel()

	change := idomain.Change{Table: "user", Operation: "UPDATE", Fields: map[string]string{
		"id": "3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
	}}

	_, err := change.Event()

	require.ErrorIs(t, err, idomain.ErrUntranslatableChange)
}

func TestChange_EventRefusesAMalformedUserID(t *testing.T) {
	t.Parallel()

	change := idomain.Change{Table: "user", Operation: "INSERT", Fields: map[string]string{"id": "ada"}}

	_, err := change.Event()

	require.ErrorIs(t, err, idomain.ErrUntranslatableChange)
}

func TestEvent_TypeIsTheFullNameOfTheMessage(t *testing.T) {
	t.Parallel()

	event := idomain.Event{Message: &events.PasswordChanged{}}

	require.Equal(t, "boilerplate.iam.v1.PasswordChanged", event.Type())
}

const (
	sessionUUID      = "8c4d2e1f-6a7b-4c3d-9e0f-1a2b3c4d5e6f"
	userUUID         = "3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b"
	organizationUUID = "3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d"
)

func TestChange_EventTranslatesAStartedSessionWithItsUserAsTheActor(t *testing.T) {
	t.Parallel()

	change := idomain.Change{
		TransactionID: "7311",
		OccurredAt:    occurredAt,
		Table:         "session",
		Operation:     "INSERT",
		Fields:        map[string]string{"id": sessionUUID, "userId": userUUID, "activeOrganizationId": organizationUUID},
		IPHash:        "5d1f0c2a",
	}

	got, err := change.Event()

	require.NoError(t, err)
	require.True(t, proto.Equal(&events.SessionStarted{
		SessionId: "session_8c4d2e1f-6a7b-4c3d-9e0f-1a2b3c4d5e6f",
		UserId:    "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
		StartedAt: timestamppb.New(occurredAt),
	}, got.Message), "got %v", got.Message)
	require.Equal(t, &idomain.Event{
		Topic:               "iam.session.v1",
		Subject:             "session_8c4d2e1f-6a7b-4c3d-9e0f-1a2b3c4d5e6f",
		Time:                occurredAt,
		CorrelationID:       "7311",
		Message:             got.Message,
		ActorID:             "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
		ActorOrganizationID: "org_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d",
		IPHash:              "5d1f0c2a",
	}, got)
}

func TestChange_EventTranslatesAStartedSessionWithoutAnActiveOrganization(t *testing.T) {
	t.Parallel()

	change := idomain.Change{Table: "session", Operation: "INSERT", Fields: map[string]string{
		"id": sessionUUID, "userId": userUUID, "activeOrganizationId": "",
	}}

	got, err := change.Event()

	require.NoError(t, err)
	require.Equal(t, "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b", got.ActorID)
	require.Empty(t, got.ActorOrganizationID)
}

func TestChange_EventTranslatesAnEndedSessionWithTheConnectionActor(t *testing.T) {
	t.Parallel()

	change := idomain.Change{
		TransactionID:       "7312",
		OccurredAt:          occurredAt,
		Table:               "session",
		Operation:           "DELETE",
		Fields:              map[string]string{"id": sessionUUID, "userId": userUUID, "activeOrganizationId": organizationUUID},
		ActorID:             uuid.NullUUID{UUID: uuid.MustParse("9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f"), Valid: true},
		ActorOrganizationID: uuid.NullUUID{UUID: uuid.MustParse(organizationUUID), Valid: true},
		IPHash:              "5d1f0c2a",
	}

	got, err := change.Event()

	require.NoError(t, err)
	require.True(t, proto.Equal(&events.SessionEnded{
		SessionId: "session_8c4d2e1f-6a7b-4c3d-9e0f-1a2b3c4d5e6f",
		UserId:    "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
		EndedAt:   timestamppb.New(occurredAt),
	}, got.Message), "got %v", got.Message)
	require.Equal(t, &idomain.Event{
		Topic:               "iam.session.v1",
		Subject:             "session_8c4d2e1f-6a7b-4c3d-9e0f-1a2b3c4d5e6f",
		Time:                occurredAt,
		CorrelationID:       "7312",
		Message:             got.Message,
		ActorID:             "user_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f",
		ActorOrganizationID: "org_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d",
		IPHash:              "5d1f0c2a",
	}, got)
}

func TestChange_EventTranslatesAnEndedSessionWithoutAConnectionActorWithItsUserAsTheActor(t *testing.T) {
	t.Parallel()

	change := idomain.Change{
		Table:     "session",
		Operation: "DELETE",
		Fields:    map[string]string{"id": sessionUUID, "userId": userUUID, "activeOrganizationId": organizationUUID},
		IPHash:    "5d1f0c2a",
	}

	got, err := change.Event()

	require.NoError(t, err)
	require.Equal(t, "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b", got.ActorID)
	require.Equal(t, "org_3f1c9a52-8d4e-4b7a-9c21-6e0f5a7b8c9d", got.ActorOrganizationID)
	require.Equal(t, "5d1f0c2a", got.IPHash)
}

func TestChange_EventTranslatesAFailedSignIn(t *testing.T) {
	t.Parallel()

	cases := map[string]events.SignInFailed_Reason{
		"invalid_credentials": events.SignInFailed_REASON_INVALID_CREDENTIALS,
		"invalid_input":       events.SignInFailed_REASON_INVALID_INPUT,
	}

	for reason, want := range cases {
		t.Run(reason, func(t *testing.T) {
			t.Parallel()

			change := idomain.Change{
				TransactionID: "7313",
				OccurredAt:    occurredAt,
				Table:         "sign_in",
				Operation:     "FAILED",
				Fields:        map[string]string{"reason": reason},
				IPHash:        "5d1f0c2a",
			}

			got, err := change.Event()

			require.NoError(t, err)
			require.True(t, proto.Equal(&events.SignInFailed{
				Reason:   want,
				FailedAt: timestamppb.New(occurredAt),
			}, got.Message), "got %v", got.Message)
			require.Equal(t, &idomain.Event{
				Topic:         "iam.session.v1",
				Time:          occurredAt,
				CorrelationID: "7313",
				Message:       got.Message,
				IPHash:        "5d1f0c2a",
			}, got)
		})
	}
}

func TestChange_EventRefusesAMalformedSessionChange(t *testing.T) {
	t.Parallel()

	cases := map[string]idomain.Change{
		"session id": {Table: "session", Operation: "INSERT", Fields: map[string]string{
			"id": "ada", "userId": userUUID,
		}},
		"user id": {Table: "session", Operation: "DELETE", Fields: map[string]string{
			"id": sessionUUID, "userId": "ada",
		}},
		"active organization id": {Table: "session", Operation: "INSERT", Fields: map[string]string{
			"id": sessionUUID, "userId": userUUID, "activeOrganizationId": "acme",
		}},
		"sign-in failure reason": {Table: "sign_in", Operation: "FAILED", Fields: map[string]string{
			"reason": "rate_limited",
		}},
	}

	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := change.Event()

			require.ErrorIs(t, err, idomain.ErrUntranslatableChange)
		})
	}
}
