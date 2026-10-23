package consumer_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
	iamevents "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/consumer"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/internal/mocks"
)

const acmeID = "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f"

func TestHandle_OrganizationDeletedDeletesTheOrganizationsNotes(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockOrganizationNotesService(gomock.NewController(t))
	service.EXPECT().DeleteOrganizationNotes(gomock.Any(), iamdomain.MustParseOrganizationID(acmeID)).Return(nil)

	err := consumer.NewHandler(service).Handle(t.Context(), eventOf(&iamevents.OrganizationDeleted{
		OrganizationId: acmeID,
		DeletedAt:      timestamppb.Now(),
	}))

	require.NoError(t, err)
}

func TestHandle_IgnoresTheOtherOrganizationEvents(t *testing.T) {
	t.Parallel()

	cases := map[string]proto.Message{
		"OrganizationCreated": &iamevents.OrganizationCreated{OrganizationId: acmeID},
		"OrganizationRenamed": &iamevents.OrganizationRenamed{OrganizationId: acmeID},
		"MemberAdded":         &iamevents.MemberAdded{OrganizationId: acmeID},
		"MemberRoleChanged":   &iamevents.MemberRoleChanged{OrganizationId: acmeID},
		"MemberRemoved":       &iamevents.MemberRemoved{OrganizationId: acmeID},
	}

	for name, msg := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			service := mocks.NewMockOrganizationNotesService(gomock.NewController(t))

			require.NoError(t, consumer.NewHandler(service).Handle(t.Context(), eventOf(msg)))
		})
	}
}

func TestHandle_IgnoresAnEventOfAnUnlinkedType(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockOrganizationNotesService(gomock.NewController(t))

	require.NoError(t, consumer.NewHandler(service).Handle(t.Context(), eventOf(nil)))
}

func TestHandle_SendsAnOrganizationDeletedWithAnInvalidIDToTheDLQ(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockOrganizationNotesService(gomock.NewController(t))

	err := consumer.NewHandler(service).Handle(t.Context(), eventOf(&iamevents.OrganizationDeleted{
		OrganizationId: "user_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f",
	}))

	require.ErrorIs(t, err, eventbus.ErrPermanent)
	require.ErrorIs(t, err, iamdomain.ErrInvalidOrganizationID)
}

func TestHandle_PassesTheServiceErrorThroughToBeRetried(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockOrganizationNotesService(gomock.NewController(t))
	service.EXPECT().DeleteOrganizationNotes(gomock.Any(), gomock.Any()).Return(errors.ErrMock)

	err := consumer.NewHandler(service).Handle(t.Context(), eventOf(&iamevents.OrganizationDeleted{
		OrganizationId: acmeID,
	}))

	require.ErrorIs(t, err, errors.ErrMock)
	require.False(t, errors.Is(err, eventbus.ErrPermanent))
}

func eventOf(msg proto.Message) *eventbus.Event {
	return (&eventbus.Event{Topic: iamevents.OrganizationTopic}).WithMessage(msg)
}
