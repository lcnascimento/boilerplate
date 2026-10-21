package consumer_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/idomain"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport/consumer"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport/internal/mocks"
)

func TestOnAny_RecordsTheEventAsAnAuditEntry(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockAuditService(gomock.NewController(t))
	service.EXPECT().Record(gomock.Any(), &idomain.Entry{
		Source:          "notes",
		ID:              "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
		Type:            "boilerplate.notes.v1.NoteCreated",
		SpecVersion:     "1.0",
		Subject:         "note_0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b00",
		Time:            time.Date(2026, 10, 17, 12, 30, 0, 0, time.UTC),
		DataContentType: "application/protobuf",
		Headers: map[string]string{
			"ce_id":     "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
			"ce_source": "notes",
		},
		Payload:        []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x0a},
		KafkaTopic:     "notes.note.v1",
		KafkaPartition: 2,
		KafkaOffset:    41,
	}).Return(nil)

	err := consumer.New(service, []string{"localhost:9092"}).OnAny(t.Context(), noteCreatedEvent())

	require.NoError(t, err)
}

func TestOnAny_RecordsTheActorOfTheEnvelope(t *testing.T) {
	t.Parallel()

	event := noteCreatedEvent()
	event.ActorID = "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b"
	event.ActorOrganizationID = "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f"

	var recorded *idomain.Entry

	service := mocks.NewMockAuditService(gomock.NewController(t))
	service.EXPECT().Record(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, entry *idomain.Entry) error {
		recorded = entry

		return nil
	})

	err := consumer.New(service, []string{"localhost:9092"}).OnAny(t.Context(), event)

	require.NoError(t, err)
	require.Equal(t, "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b", recorded.ActorID)
	require.Equal(t, "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f", recorded.ActorOrganizationID)
}

func TestOnAny_PassesTheServiceErrorThrough(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		serviceErr error
		permanent  bool
	}{
		"transient error is retried": {serviceErr: errors.ErrMock, permanent: false},
		"permanent error goes to the DLQ": {
			serviceErr: events.ErrPermanent.WithCause(errors.ErrMock),
			permanent:  true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			service := mocks.NewMockAuditService(gomock.NewController(t))
			service.EXPECT().Record(gomock.Any(), gomock.Any()).Return(tc.serviceErr)

			err := consumer.New(service, []string{"localhost:9092"}).OnAny(t.Context(), noteCreatedEvent())

			require.ErrorIs(t, err, errors.ErrMock)
			require.Equal(t, tc.permanent, errors.Is(err, events.ErrPermanent))
		})
	}
}

func TestReady_FailsBeforeStart(t *testing.T) {
	t.Parallel()

	service := mocks.NewMockAuditService(gomock.NewController(t))

	require.ErrorIs(t, consumer.New(service, []string{"localhost:9092"}).Ready(t.Context()), consumer.ErrNotConsuming)
}

func noteCreatedEvent() *events.Event {
	return &events.Event{
		Envelope: events.Envelope{
			ID:              "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
			Source:          "notes",
			Type:            "boilerplate.notes.v1.NoteCreated",
			Time:            time.Date(2026, 10, 17, 12, 30, 0, 0, time.UTC),
			Subject:         "note_0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b00",
			PartitionKey:    "note_0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b00",
			SpecVersion:     "1.0",
			DataContentType: "application/protobuf",
		},
		Headers: map[string]string{
			"ce_id":     "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b5e",
			"ce_source": "notes",
		},
		Payload:   []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x0a},
		SchemaID:  1,
		Topic:     "notes.note.v1",
		Partition: 2,
		Offset:    41,
	}
}
