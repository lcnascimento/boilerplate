package events_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	kiterrors "github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events/eventstest"
)

const waitFor = 30 * time.Second

func TestDLQTopic(t *testing.T) {
	t.Parallel()

	require.Equal(t, "notes.note.v1.audit.dlq", events.DLQTopic(events.Topic, "audit"))
}

func TestNewSubscriber_InvalidConfig(t *testing.T) {
	t.Parallel()

	valid := events.SubscriberConfig{Brokers: []string{"localhost:9092"}, Group: "audit", Observer: newRecorder()}

	tests := []struct {
		name   string
		mutate func(*events.SubscriberConfig)
	}{
		{name: "no brokers", mutate: func(c *events.SubscriberConfig) { c.Brokers = nil }},
		{name: "no group", mutate: func(c *events.SubscriberConfig) { c.Group = "" }},
		{name: "no observer", mutate: func(c *events.SubscriberConfig) { c.Observer = nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := valid
			tt.mutate(&cfg)

			_, err := events.NewSubscriber(cfg)

			require.ErrorIs(t, err, events.ErrInvalidSubscriberConfig)
		})
	}
}

func TestComponentSubscriber_DeliversTypedThenOnAny(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := eventstest.New(t)
	sent := &events.NoteCreated{Id: "note_1", Title: "t", Body: "b", CreatedAt: timestamppb.New(time.Unix(1700000000, 0))}
	envelope := env.Publish(t, "note_1", sent)

	observer := newRecorder()
	startSubscriber(t, env, observer)

	typed := observer.next(t)
	anyCall := observer.next(t)

	require.Equal(t, "OnNoteCreated", typed.method)
	require.Equal(t, "OnAny", anyCall.method)
	require.True(t, proto.Equal(sent, typed.typed), "want %v, got %v", sent, typed.typed)

	event := anyCall.event
	require.Equal(t, envelope.ID, event.ID)
	require.Equal(t, "eventstest", event.Source)
	require.Equal(t, "boilerplate.notes.v1.NoteCreated", event.Type)
	require.Equal(t, "note_1", event.Subject)
	require.Equal(t, "note_1", event.PartitionKey)
	require.True(t, envelope.Time.Equal(event.Time), "want %v, got %v", envelope.Time, event.Time)
	require.Equal(t, envelope.ID, event.Headers["ce_id"])
	require.Equal(t, "boilerplate.notes.v1.NoteCreated", event.Headers["ce_type"])
	require.Equal(t, envelope.ID, event.Headers["_watermill_message_uuid"])
	require.Equal(t, env.Topic, event.Topic)
	require.Equal(t, int64(0), event.Offset)
	require.Contains(t, []int32{0, 1, 2}, event.Partition)
	require.True(t, proto.Equal(sent, event.Message), "want %v, got %v", sent, event.Message)

	schemaID, decoded, err := events.Decode(event.Type, event.Payload)
	require.NoError(t, err)
	require.Positive(t, event.SchemaID)
	require.Equal(t, schemaID, event.SchemaID)
	require.True(t, proto.Equal(sent, decoded))
}

func TestComponentSubscriber_UnknownTypeReachesOnlyOnAny(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := eventstest.New(t)
	headers := validHeaders("boilerplate.notes.v1.NoteArchived")
	env.PublishRaw(t, headers["ce_id"], headers, []byte{0x00, 0x00, 0x00, 0x00, 0x2a, 0x00, 0x0a, 0x01, 'x'})

	observer := newRecorder()
	startSubscriber(t, env, observer)

	got := observer.next(t)

	require.Equal(t, "OnAny", got.method)
	require.Equal(t, "boilerplate.notes.v1.NoteArchived", got.event.Type)
	require.Equal(t, 42, got.event.SchemaID)
	require.Nil(t, got.event.Message)
	observer.requireNoMoreCalls(t)
}

func TestComponentSubscriber_NewGroupConsumesFromOldestOffset(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := eventstest.New(t)
	first := env.Publish(t, "note_1", &events.NoteDeleted{Id: "note_1"})
	second := env.Publish(t, "note_1", &events.NoteDeleted{Id: "note_1"})

	observer := newRecorder()
	observer.typedCalls = false
	startSubscriber(t, env, observer)

	require.Equal(t, first.ID, observer.next(t).event.ID)
	require.Equal(t, second.ID, observer.next(t).event.ID)
}

func TestComponentSubscriber_TransientErrorBlocksThePartitionUntilItSucceeds(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := eventstest.New(t)
	first := env.Publish(t, "note_1", &events.NoteDeleted{Id: "note_1"})
	second := env.Publish(t, "note_1", &events.NoteDeleted{Id: "note_1"})
	third := env.Publish(t, "note_1", &events.NoteDeleted{Id: "note_1"})

	var failures atomic.Int32

	observer := newRecorder()
	observer.typedCalls = false
	observer.onAny = func(event *events.Event) error {
		if event.ID == first.ID && failures.Add(1) <= 3 {
			return errors.New("database is down")
		}

		return nil
	}
	startSubscriber(t, env, observer)

	require.Equal(t, first.ID, observer.next(t).event.ID)
	require.Equal(t, second.ID, observer.next(t).event.ID)
	require.Equal(t, third.ID, observer.next(t).event.ID)
	require.Equal(t, int32(4), failures.Load())
}

func TestComponentSubscriber_PermanentErrorGoesToTheDLQ(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("component test")
	}

	tests := []struct {
		name       string
		onAny      func() error
		wantReason string
	}{
		{
			name:       "ErrPermanent",
			onAny:      func() error { return events.ErrPermanent.WithCause(errors.New("note is corrupt")) },
			wantReason: "message cannot be processed and goes to the DLQ",
		},
		{
			name:       "panic",
			onAny:      func() error { panic("observer bug") },
			wantReason: "panic occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := eventstest.New(t)
			poisoned := env.Publish(t, "note_1", &events.NoteDeleted{Id: "note_1"})
			next := env.Publish(t, "note_1", &events.NoteDeleted{Id: "note_1"})

			var calls atomic.Int32

			observer := newRecorder()
			observer.typedCalls = false
			observer.onAny = func(event *events.Event) error {
				if event.ID != poisoned.ID {
					return nil
				}

				calls.Add(1)

				return tt.onAny()
			}
			startSubscriber(t, env, observer)

			require.Equal(t, next.ID, observer.next(t).event.ID)
			require.Equal(t, int32(1), calls.Load())

			record := consumeDLQ(t, env, 1)[0]
			headers := headersOf(record)

			require.Equal(t, "note_1", string(record.Key))
			require.Equal(t, poisoned.ID, headers["ce_id"])
			require.Equal(t, poisoned.ID, headers["_watermill_message_uuid"])
			require.Equal(t, "boilerplate.notes.v1.NoteDeleted", headers["ce_type"])
			require.Equal(t, "note_1", headers["ce_partitionkey"])
			require.Contains(t, headers[middleware.ReasonForPoisonedKey], tt.wantReason)
			require.Equal(t, env.Topic, headers[middleware.PoisonedTopicKey])

			_, message, err := events.Decode("boilerplate.notes.v1.NoteDeleted", record.Value)
			require.NoError(t, err)
			require.True(t, proto.Equal(&events.NoteDeleted{Id: "note_1"}, message))
		})
	}
}

func TestComponentSubscriber_MalformedMessageGoesToTheDLQWithoutCallingTheObserver(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("component test")
	}

	tests := []struct {
		name       string
		headers    map[string]string
		value      []byte
		wantReason string
	}{
		{
			name:       "wrong magic byte",
			headers:    validHeaders("boilerplate.notes.v1.NoteDeleted"),
			value:      []byte{0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x0a, 0x01, 'x'},
			wantReason: "Confluent wire format",
		},
		{
			name:       "truncated schema ID",
			headers:    validHeaders("boilerplate.notes.v1.NoteDeleted"),
			value:      []byte{0x00, 0x00, 0x01},
			wantReason: "Confluent wire format",
		},
		{
			name:       "payload that does not decode under any schema",
			headers:    validHeaders("boilerplate.notes.v1.NoteDeleted"),
			value:      []byte{0x00, 0x7f, 0xff, 0xff, 0xff, 0x00, 0xff, 0xff},
			wantReason: "not a valid Protobuf message",
		},
		{
			name:       "missing CloudEvents header",
			headers:    withoutHeader(validHeaders("boilerplate.notes.v1.NoteDeleted"), "ce_type"),
			value:      []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x0a, 0x01, 'x'},
			wantReason: "required CloudEvents header is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := eventstest.New(t)
			env.PublishRaw(t, tt.headers["ce_id"], tt.headers, tt.value)
			next := env.Publish(t, "note_1", &events.NoteDeleted{Id: "note_1"})

			observer := newRecorder()
			observer.typedCalls = false
			startSubscriber(t, env, observer)

			require.Equal(t, next.ID, observer.next(t).event.ID)

			record := consumeDLQ(t, env, 1)[0]
			headers := headersOf(record)

			require.Equal(t, tt.value, record.Value)
			require.Equal(t, tt.headers["ce_id"], headers["ce_id"])
			require.Contains(t, headers[middleware.ReasonForPoisonedKey], tt.wantReason)
			observer.requireNoMoreCalls(t)
		})
	}
}

func TestComponentSubscriber_StartFailsWhenATopicIsMissing(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("component test")
	}

	t.Run("source topic", func(t *testing.T) {
		t.Parallel()

		env := eventstest.New(t)
		env.Topic = eventstest.RandomName()
		env.CreateTopic(t, events.DLQTopic(env.Topic, env.Group), 1)

		_, err := newSubscriber(t, env, newRecorder())

		require.ErrorIs(t, err, events.ErrTopicMissing)
		require.Equal(t, "TOPIC_MISSING", string(kiterrors.Code(err)))
	})

	t.Run("DLQ", func(t *testing.T) {
		t.Parallel()

		env := eventstest.New(t)

		_, err := newSubscriber(t, env, newRecorder())

		require.ErrorIs(t, err, events.ErrTopicMissing)
	})
}

func withoutHeader(headers map[string]string, key string) map[string]string {
	delete(headers, key)

	return headers
}

func validHeaders(eventType string) map[string]string {
	envelope := events.Envelope{
		ID:           uuid.Must(uuid.NewV7()).String(),
		Source:       "test",
		Type:         eventType,
		PartitionKey: "note_1",
		SpecVersion:  events.SpecVersion,
	}

	return envelope.Headers()
}

type call struct {
	method string
	event  *events.Event
	typed  proto.Message
}

type recorder struct {
	events.UnimplementedObserver

	calls      chan call
	typedCalls bool
	onAny      func(*events.Event) error
}

func newRecorder() *recorder {
	return &recorder{calls: make(chan call, 100), typedCalls: true}
}

func (r *recorder) OnNoteCreated(_ context.Context, event *events.Event, note *events.NoteCreated) error {
	if r.typedCalls {
		r.calls <- call{method: "OnNoteCreated", event: event, typed: note}
	}

	return nil
}

func (r *recorder) OnNoteDeleted(_ context.Context, event *events.Event, note *events.NoteDeleted) error {
	if r.typedCalls {
		r.calls <- call{method: "OnNoteDeleted", event: event, typed: note}
	}

	return nil
}

func (r *recorder) OnAny(_ context.Context, event *events.Event) error {
	if r.onAny != nil {
		if err := r.onAny(event); err != nil {
			return err
		}
	}

	r.calls <- call{method: "OnAny", event: event}

	return nil
}

func (r *recorder) next(t *testing.T) call {
	t.Helper()

	select {
	case c := <-r.calls:
		return c
	case <-time.After(waitFor):
		t.Fatal("observer was not called")

		return call{}
	}
}

func (r *recorder) requireNoMoreCalls(t *testing.T) {
	t.Helper()

	select {
	case c := <-r.calls:
		t.Fatalf("unexpected %s call for %s", c.method, c.event.Type)
	case <-time.After(time.Second):
	}
}

func startSubscriber(t *testing.T, env *eventstest.Env, observer events.Observer) {
	t.Helper()

	env.CreateTopic(t, events.DLQTopic(env.Topic, env.Group), 1)

	subscriber, err := newSubscriber(t, env, observer)
	require.NoError(t, err)

	done := make(chan error, 1)

	go func() { done <- subscriber.Run(t.Context()) }()

	t.Cleanup(func() {
		require.NoError(t, subscriber.Close())
		require.NoError(t, <-done)
	})

	select {
	case <-subscriber.Running():
	case err := <-done:
		t.Fatalf("subscriber stopped before running: %v", err)
	case <-time.After(waitFor):
		t.Fatal("subscriber did not start")
	}
}

func newSubscriber(t *testing.T, env *eventstest.Env, observer events.Observer) (*events.Subscriber, error) {
	t.Helper()

	return events.NewSubscriberOfTopic(events.SubscriberConfig{
		Brokers:  env.Brokers,
		Group:    env.Group,
		Observer: observer,
	}, env.Topic)
}

func consumeDLQ(t *testing.T, env *eventstest.Env, n int) []*sarama.ConsumerMessage {
	t.Helper()

	consumer, err := sarama.NewConsumer(env.Brokers, sarama.NewConfig())
	require.NoError(t, err)

	defer func() { _ = consumer.Close() }()

	partition, err := consumer.ConsumePartition(events.DLQTopic(env.Topic, env.Group), 0, sarama.OffsetOldest)
	require.NoError(t, err)

	defer func() { _ = partition.Close() }()

	var records []*sarama.ConsumerMessage

	for len(records) < n {
		select {
		case record := <-partition.Messages():
			records = append(records, record)
		case <-time.After(waitFor):
			t.Fatalf("got %d DLQ records, want %d", len(records), n)
		}
	}

	return records
}

func headersOf(record *sarama.ConsumerMessage) map[string]string {
	headers := map[string]string{}
	for _, h := range record.Headers {
		headers[string(h.Key)] = string(h.Value)
	}

	return headers
}
