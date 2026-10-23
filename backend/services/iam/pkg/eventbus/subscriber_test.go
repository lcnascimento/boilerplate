package eventbus_test

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	kiterrors "github.com/lcnascimento/go-kit/errors"
	otelbaggage "go.opentelemetry.io/otel/baggage"

	"github.com/lcnascimento/go-kit/o11y/baggage"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/eventbustest"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/internal/testevents"
)

var (
	thingCreatedDescriptor = (&testevents.ThingCreated{}).ProtoReflect().Descriptor()
	thingCreatedType       = string(thingCreatedDescriptor.FullName())
	thingDeletedType       = string((&testevents.ThingDeleted{}).ProtoReflect().Descriptor().FullName())
	thingArchivedType      = string(thingCreatedDescriptor.FullName().Parent().Append("ThingArchived"))
)

const waitFor = 30 * time.Second

func TestMain(m *testing.M) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	os.Exit(m.Run())
}

func TestDLQTopic(t *testing.T) {
	t.Parallel()

	require.Equal(t, "iam.user.v1.audit.dlq", eventbus.DLQTopic("iam.user.v1", "audit"))
}

func TestNewSubscriber_InvalidConfig(t *testing.T) {
	t.Parallel()

	valid := eventbus.SubscriberConfig{
		Brokers: []string{"localhost:9092"},
		Group:   "audit",
		Topics:  []string{"iam.user.v1"},
		Handler: newRecorder(),
	}

	tests := []struct {
		name   string
		mutate func(*eventbus.SubscriberConfig)
	}{
		{name: "no brokers", mutate: func(c *eventbus.SubscriberConfig) { c.Brokers = nil }},
		{name: "no group", mutate: func(c *eventbus.SubscriberConfig) { c.Group = "" }},
		{name: "no topics", mutate: func(c *eventbus.SubscriberConfig) { c.Topics = nil }},
		{name: "no handler", mutate: func(c *eventbus.SubscriberConfig) { c.Handler = nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := valid
			tt.mutate(&cfg)

			_, err := eventbus.NewSubscriber(&cfg)

			require.ErrorIs(t, err, eventbus.ErrInvalidSubscriberConfig)
		})
	}
}

func TestNewSubscriber_NilConfig(t *testing.T) {
	t.Parallel()

	_, err := eventbus.NewSubscriber(nil)

	require.ErrorIs(t, err, eventbus.ErrInvalidSubscriberConfig)
}

func TestComponentSubscriber_DeliversTheDecodedEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := eventbustest.New(t, testevents.Protos)
	sent := &testevents.ThingCreated{Id: "thing_1", Title: "t", CreatedAt: timestamppb.New(time.Unix(1700000000, 0))}
	envelope := env.Publish(t, "thing_1", sent)

	handler := newRecorder()
	startSubscriber(t, env, handler)

	event := handler.next(t)

	require.Equal(t, envelope.ID, event.ID)
	require.Equal(t, "eventbustest", event.Source)
	require.Equal(t, thingCreatedType, event.Type)
	require.Equal(t, "thing_1", event.Subject)
	require.Equal(t, "thing_1", event.PartitionKey)
	require.True(t, envelope.Time.Equal(event.Time), "want %v, got %v", envelope.Time, event.Time)
	require.Equal(t, envelope.ID, event.Headers["ce_id"])
	require.Equal(t, thingCreatedType, event.Headers["ce_type"])
	require.Equal(t, envelope.ID, event.Headers["_watermill_message_uuid"])
	require.Equal(t, env.Topic, event.Topic)
	require.Equal(t, int64(0), event.Offset)
	require.Contains(t, []int32{0, 1, 2}, event.Partition)
	got, ok := event.Message()
	require.True(t, ok)
	require.True(t, proto.Equal(sent, got), "want %v, got %v", sent, got)

	schemaID, decoded, err := eventbus.Decode(event.Type, event.Payload)
	require.NoError(t, err)
	require.Positive(t, event.SchemaID)
	require.Equal(t, schemaID, event.SchemaID)
	require.True(t, proto.Equal(sent, decoded))
}

func TestComponentSubscriber_HandlesTheEventWithTheProducerBaggage(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := eventbustest.New(t, testevents.Protos)
	producer := baggage.ContextWithMembers(t.Context(),
		baggage.NewMember("user.id", "user_1"),
		baggage.NewMember("organization.id", "org_1"),
	)
	env.PublishFrom(producer, t, "thing_1", &testevents.ThingCreated{Id: "thing_1"})

	handler := baggageRecorder(make(chan otelbaggage.Baggage, 1))
	startSubscriber(t, env, handler)

	select {
	case bag := <-handler:
		require.Equal(t, "user_1", bag.Member("user.id").Value())
		require.Equal(t, "org_1", bag.Member("organization.id").Value())
	case <-time.After(waitFor):
		t.Fatal("handler was not called")
	}
}

func TestComponentSubscriber_DeliversEveryTopicOfTheGroup(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := eventbustest.New(t, testevents.Protos)
	other := eventbustest.RandomName()
	env.CreateTopic(t, other, 1)

	first := env.Publish(t, "thing_1", &testevents.ThingDeleted{Id: "thing_1"})
	second := env.PublishTo(t, other, "thing_2", &testevents.ThingDeleted{Id: "thing_2"})

	handler := newRecorder()
	startSubscriber(t, env, handler, other)

	got := map[string]string{}

	for range 2 {
		event := handler.next(t)
		got[event.ID] = event.Topic
	}

	require.Equal(t, map[string]string{first.ID: env.Topic, second.ID: other}, got)
}

func TestComponentSubscriber_DeliversAnUnknownTypeWithoutMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := eventbustest.New(t, testevents.Protos)
	headers := validHeaders(thingArchivedType)
	env.PublishRaw(t, headers["ce_id"], headers, []byte{0x00, 0x00, 0x00, 0x00, 0x2a, 0x00, 0x0a, 0x01, 'x'})

	handler := newRecorder()
	startSubscriber(t, env, handler)

	got := handler.next(t)

	require.Equal(t, thingArchivedType, got.Type)
	require.Equal(t, 42, got.SchemaID)
	_, decoded := got.Message()
	require.False(t, decoded)
	handler.requireNoMoreCalls(t)
}

func TestComponentSubscriber_NewGroupConsumesFromOldestOffset(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := eventbustest.New(t, testevents.Protos)
	first := env.Publish(t, "thing_1", &testevents.ThingDeleted{Id: "thing_1"})
	second := env.Publish(t, "thing_1", &testevents.ThingDeleted{Id: "thing_1"})

	handler := newRecorder()
	startSubscriber(t, env, handler)

	require.Equal(t, first.ID, handler.next(t).ID)
	require.Equal(t, second.ID, handler.next(t).ID)
}

func TestComponentSubscriber_TransientErrorBlocksThePartitionUntilItSucceeds(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := eventbustest.New(t, testevents.Protos)
	first := env.Publish(t, "thing_1", &testevents.ThingDeleted{Id: "thing_1"})
	second := env.Publish(t, "thing_1", &testevents.ThingDeleted{Id: "thing_1"})
	third := env.Publish(t, "thing_1", &testevents.ThingDeleted{Id: "thing_1"})

	var failures atomic.Int32

	handler := newRecorder()
	handler.onHandle = func(event *eventbus.Event) error {
		if event.ID == first.ID && failures.Add(1) <= 3 {
			return errors.New("database is down")
		}

		return nil
	}
	startSubscriber(t, env, handler)

	require.Equal(t, first.ID, handler.next(t).ID)
	require.Equal(t, second.ID, handler.next(t).ID)
	require.Equal(t, third.ID, handler.next(t).ID)
	require.Equal(t, int32(4), failures.Load())
}

func TestComponentSubscriber_PermanentErrorGoesToTheDLQ(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("component test")
	}

	tests := []struct {
		name       string
		handle     func() error
		wantReason string
	}{
		{
			name:       "ErrPermanent",
			handle:     func() error { return eventbus.ErrPermanent.WithCause(errors.New("payload is corrupt")) },
			wantReason: "message cannot be processed and goes to the DLQ",
		},
		{
			name:       "panic",
			handle:     func() error { panic("handler bug") },
			wantReason: "panic occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := eventbustest.New(t, testevents.Protos)
			poisoned := env.Publish(t, "thing_1", &testevents.ThingDeleted{Id: "thing_1"})
			next := env.Publish(t, "thing_1", &testevents.ThingDeleted{Id: "thing_1"})

			var calls atomic.Int32

			handler := newRecorder()
			handler.onHandle = func(event *eventbus.Event) error {
				if event.ID != poisoned.ID {
					return nil
				}

				calls.Add(1)

				return tt.handle()
			}
			startSubscriber(t, env, handler)

			require.Equal(t, next.ID, handler.next(t).ID)
			require.Equal(t, int32(1), calls.Load())

			record := consumeDLQ(t, env, 1)[0]
			headers := headersOf(record)

			require.Equal(t, "thing_1", string(record.Key))
			require.Equal(t, poisoned.ID, headers["ce_id"])
			require.Equal(t, poisoned.ID, headers["_watermill_message_uuid"])
			require.Equal(t, thingDeletedType, headers["ce_type"])
			require.Equal(t, "thing_1", headers["ce_partitionkey"])
			require.Contains(t, headers[middleware.ReasonForPoisonedKey], tt.wantReason)
			require.Equal(t, env.Topic, headers[middleware.PoisonedTopicKey])

			_, message, err := eventbus.Decode(thingDeletedType, record.Value)
			require.NoError(t, err)
			require.True(t, proto.Equal(&testevents.ThingDeleted{Id: "thing_1"}, message))
		})
	}
}

func TestComponentSubscriber_MalformedMessageGoesToTheDLQWithoutCallingTheHandler(t *testing.T) {
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
			headers:    validHeaders(thingDeletedType),
			value:      []byte{0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x0a, 0x01, 'x'},
			wantReason: "Confluent wire format",
		},
		{
			name:       "truncated schema ID",
			headers:    validHeaders(thingDeletedType),
			value:      []byte{0x00, 0x00, 0x01},
			wantReason: "Confluent wire format",
		},
		{
			name:       "payload that does not decode under any schema",
			headers:    validHeaders(thingDeletedType),
			value:      []byte{0x00, 0x7f, 0xff, 0xff, 0xff, 0x00, 0xff, 0xff},
			wantReason: "not a valid Protobuf message",
		},
		{
			name:       "missing CloudEvents header",
			headers:    withoutHeader(validHeaders(thingDeletedType), "ce_type"),
			value:      []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x0a, 0x01, 'x'},
			wantReason: "required CloudEvents header is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := eventbustest.New(t, testevents.Protos)
			env.PublishRaw(t, tt.headers["ce_id"], tt.headers, tt.value)
			next := env.Publish(t, "thing_1", &testevents.ThingDeleted{Id: "thing_1"})

			handler := newRecorder()
			startSubscriber(t, env, handler)

			require.Equal(t, next.ID, handler.next(t).ID)

			record := consumeDLQ(t, env, 1)[0]
			headers := headersOf(record)

			require.Equal(t, tt.value, record.Value)
			require.Equal(t, tt.headers["ce_id"], headers["ce_id"])
			require.Contains(t, headers[middleware.ReasonForPoisonedKey], tt.wantReason)
			handler.requireNoMoreCalls(t)
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

		env := eventbustest.New(t, testevents.Protos)
		env.Topic = eventbustest.RandomName()
		env.CreateTopic(t, eventbus.DLQTopic(env.Topic, env.Group), 1)

		_, err := newSubscriber(t, env, newRecorder())

		require.ErrorIs(t, err, eventbus.ErrTopicMissing)
		require.Equal(t, "TOPIC_MISSING", string(kiterrors.Code(err)))
	})

	t.Run("DLQ", func(t *testing.T) {
		t.Parallel()

		env := eventbustest.New(t, testevents.Protos)

		_, err := newSubscriber(t, env, newRecorder())

		require.ErrorIs(t, err, eventbus.ErrTopicMissing)
	})
}

func withoutHeader(headers map[string]string, key string) map[string]string {
	delete(headers, key)

	return headers
}

func validHeaders(eventType string) map[string]string {
	envelope := eventbus.Envelope{
		ID:           uuid.Must(uuid.NewV7()).String(),
		Source:       "test",
		Type:         eventType,
		PartitionKey: "thing_1",
		SpecVersion:  eventbus.SpecVersion,
	}

	return envelope.Headers()
}

type baggageRecorder chan otelbaggage.Baggage

func (r baggageRecorder) Handle(ctx context.Context, _ *eventbus.Event) error {
	r <- otelbaggage.FromContext(ctx)

	return nil
}

type recorder struct {
	events   chan *eventbus.Event
	onHandle func(*eventbus.Event) error
}

func newRecorder() *recorder {
	return &recorder{events: make(chan *eventbus.Event, 100)}
}

func (r *recorder) Handle(_ context.Context, event *eventbus.Event) error {
	if r.onHandle != nil {
		if err := r.onHandle(event); err != nil {
			return err
		}
	}

	r.events <- event

	return nil
}

func (r *recorder) next(t *testing.T) *eventbus.Event {
	t.Helper()

	select {
	case event := <-r.events:
		return event
	case <-time.After(waitFor):
		t.Fatal("handler was not called")

		return nil
	}
}

func (r *recorder) requireNoMoreCalls(t *testing.T) {
	t.Helper()

	select {
	case event := <-r.events:
		t.Fatalf("unexpected call for %s", event.Type)
	case <-time.After(time.Second):
	}
}

func startSubscriber(t *testing.T, env *eventbustest.Env, handler eventbus.Handler, otherTopics ...string) {
	t.Helper()

	for _, topic := range append([]string{env.Topic}, otherTopics...) {
		env.CreateTopic(t, eventbus.DLQTopic(topic, env.Group), 1)
	}

	subscriber, err := newSubscriber(t, env, handler, otherTopics...)
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

func newSubscriber(
	t *testing.T, env *eventbustest.Env, handler eventbus.Handler, otherTopics ...string,
) (*eventbus.Subscriber, error) {
	t.Helper()

	return eventbus.NewSubscriber(&eventbus.SubscriberConfig{
		Brokers: env.Brokers,
		Group:   env.Group,
		Topics:  append([]string{env.Topic}, otherTopics...),
		Handler: handler,
	})
}

func consumeDLQ(t *testing.T, env *eventbustest.Env, n int) []*sarama.ConsumerMessage {
	t.Helper()

	consumer, err := sarama.NewConsumer(env.Brokers, sarama.NewConfig())
	require.NoError(t, err)

	defer func() { _ = consumer.Close() }()

	partition, err := consumer.ConsumePartition(eventbus.DLQTopic(env.Topic, env.Group), 0, sarama.OffsetOldest)
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
