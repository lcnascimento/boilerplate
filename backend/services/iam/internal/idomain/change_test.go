package idomain_test

import (
	"testing"
	"time"

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
