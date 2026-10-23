package consumer_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/idomain"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport/consumer"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport/internal/mocks"
)

func TestHandle_RecordsAnIamEventAsAnAuditEntry(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockAuditService(gomock.NewController(t))
	service.EXPECT().Record(gomock.Any(), &idomain.Entry{
		Source:          "iam",
		ID:              "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
		Type:            "boilerplate.iam.v1.MemberRoleChanged",
		SpecVersion:     "1.0",
		Subject:         "member_0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b00",
		Time:            time.Date(2026, 10, 17, 12, 30, 0, 0, time.UTC),
		DataContentType: "application/protobuf",
		Headers: map[string]string{
			"ce_id":     "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
			"ce_source": "iam",
		},
		Payload:        []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x0a},
		KafkaTopic:     "iam.organization.v1",
		KafkaPartition: 2,
		KafkaOffset:    41,

		ActorID:             "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
		ActorOrganizationID: "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f",
		IPHash:              "5d41402abc4b2a76b9719d911017c592ae0b6a0f7d5b3e1c2a4f6e8d0b1c3a5f",
		CorrelationID:       "784512",
	}).Return(nil)

	err := consumer.New(service, []string{"localhost:9092"}).Handle(t.Context(), memberRoleChangedEvent())

	require.NoError(t, err)
}

func TestHandle_RecordsAnEnvelopeWithoutActorIPHashOrCorrelationWithThemEmpty(t *testing.T) {
	t.Parallel()

	event := memberRoleChangedEvent()
	event.ActorID = ""
	event.ActorOrganizationID = ""
	event.IPHash = ""
	event.CorrelationID = ""

	var recorded *idomain.Entry

	service := mocks.NewMockAuditService(gomock.NewController(t))
	service.EXPECT().Record(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, entry *idomain.Entry) error {
		recorded = entry

		return nil
	})

	err := consumer.New(service, []string{"localhost:9092"}).Handle(t.Context(), event)

	require.NoError(t, err)
	require.Equal(t, "boilerplate.iam.v1.MemberRoleChanged", recorded.Type)
	require.Empty(t, recorded.ActorID)
	require.Empty(t, recorded.ActorOrganizationID)
	require.Empty(t, recorded.IPHash)
	require.Empty(t, recorded.CorrelationID)
}

func TestHandle_PassesTheServiceErrorThrough(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		serviceErr error
		permanent  bool
	}{
		"transient error is retried": {serviceErr: errors.ErrMock, permanent: false},
		"permanent error goes to the DLQ": {
			serviceErr: eventbus.ErrPermanent.WithCause(errors.ErrMock),
			permanent:  true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			service := mocks.NewMockAuditService(gomock.NewController(t))
			service.EXPECT().Record(gomock.Any(), gomock.Any()).Return(tc.serviceErr)

			err := consumer.New(service, []string{"localhost:9092"}).Handle(t.Context(), memberRoleChangedEvent())

			require.ErrorIs(t, err, errors.ErrMock)
			require.Equal(t, tc.permanent, errors.Is(err, eventbus.ErrPermanent))
		})
	}
}

func TestReady_FailsBeforeStart(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockAuditService(gomock.NewController(t))

	require.ErrorIs(t, consumer.New(service, []string{"localhost:9092"}).Ready(t.Context()), consumer.ErrNotConsuming)
}

func memberRoleChangedEvent() *eventbus.Event {
	return &eventbus.Event{
		Envelope: eventbus.Envelope{
			ID:                  "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
			Source:              "iam",
			Type:                "boilerplate.iam.v1.MemberRoleChanged",
			Time:                time.Date(2026, 10, 17, 12, 30, 0, 0, time.UTC),
			Subject:             "member_0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b00",
			PartitionKey:        "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f",
			SpecVersion:         "1.0",
			DataContentType:     "application/protobuf",
			ActorID:             "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b",
			ActorOrganizationID: "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f",
			IPHash:              "5d41402abc4b2a76b9719d911017c592ae0b6a0f7d5b3e1c2a4f6e8d0b1c3a5f",
			CorrelationID:       "784512",
		},
		Headers: map[string]string{
			"ce_id":     "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
			"ce_source": "iam",
		},
		Payload:   []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x0a},
		SchemaID:  1,
		Topic:     "iam.organization.v1",
		Partition: 2,
		Offset:    41,
	}
}
