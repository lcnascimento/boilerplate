package forwarder_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/lcnascimento/go-kit/outbox"
	"github.com/lcnascimento/go-kit/postgres/testdb"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/eventbustest"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/roles"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/migrations"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/repository/notes"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/worker/forwarder"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

var (
	noteCreatedDescriptor = (&events.NoteCreated{}).ProtoReflect().Descriptor()
	noteCreatedType       = string(noteCreatedDescriptor.FullName())
	noteDeletedType       = string((&events.NoteDeleted{}).ProtoReflect().Descriptor().FullName())
)

const waitFor = 30 * time.Second

var author = &actor.Actor{
	UserID:         mustParseUserID("user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b"),
	OrganizationID: iamdomain.MustParseOrganizationID("org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f"),
	Role:           roles.Admin,
}

var (
	spans   = tracetest.NewSpanRecorder()
	metrics = sdkmetric.NewManualReader()
)

func TestMain(m *testing.M) {
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans)))
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(metrics)))
	otel.SetTextMapPropagator(propagation.TraceContext{})

	os.Exit(m.Run())
}

func TestComponentForwarder_DeliversCreateAndDeleteToTheEventsSubscriber(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := newEnv(t)
	deletedAt := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	repo := notes.New(env.pool, env.schemaIDs, notes.WithClock(func() time.Time { return deletedAt }))
	note := newNote("Lista de compras", time.Date(2026, 10, 10, 19, 30, 0, 123456000, time.UTC))

	env.startForwarder(t)
	require.NoError(t, repo.Create(t.Context(), author, note))
	require.NoError(t, repo.Delete(t.Context(), author, note.ID))

	observer := env.subscribe(t, note.ID)
	created := observer.next(t)
	deleted := observer.next(t)

	wantCreated := &events.NoteCreated{
		Id:             note.ID.String(),
		OrganizationId: "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f",
		Title:          "Lista de compras",
		Body:           "body of Lista de compras",
		CreatedAt:      timestamppb.New(time.Date(2026, 10, 10, 19, 30, 0, 123456000, time.UTC)),
	}
	require.True(t, proto.Equal(wantCreated, created.Message), "want %v, got %v", wantCreated, created.Message)
	require.Equal(t, noteCreatedType, created.Type)
	require.Equal(t, "notes", created.Source)
	require.Equal(t, note.ID.String(), created.Subject)
	require.Equal(t, events.Topic, created.Topic)
	require.True(t, note.CreatedAt.Equal(created.Time), "want %v, got %v", note.CreatedAt, created.Time)
	require.Equal(t, created.ID, created.Headers["_watermill_message_uuid"])
	require.Equal(t, "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b", created.ActorID)
	require.Equal(t, "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f", created.ActorOrganizationID)

	wantDeleted := &events.NoteDeleted{Id: note.ID.String(), OrganizationId: "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f"}
	require.True(t, proto.Equal(wantDeleted, deleted.Message), "want %v, got %v", wantDeleted, deleted.Message)
	require.Equal(t, noteDeletedType, deleted.Type)
	require.True(t, deletedAt.Equal(deleted.Time), "want %v, got %v", deletedAt, deleted.Time)
	require.NotEqual(t, created.ID, deleted.ID)
	require.Equal(t, "user_3f2a1b4c-5d6e-4f70-8a9b-0c1d2e3f4a5b", deleted.ActorID)
	require.Equal(t, "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f", deleted.ActorOrganizationID)
}

func TestComponentForwarder_KeepsTheOrderOfANoteUnderItsIDAsKey(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := newEnv(t)
	repo := notes.New(env.pool, env.schemaIDs)
	note := newNote("recreated", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))

	for range 3 {
		require.NoError(t, repo.Create(t.Context(), author, note))
		require.NoError(t, repo.Delete(t.Context(), author, note.ID))
	}

	env.startForwarder(t)

	records := consume(t, env.brokers, note.ID, 6)

	wantTypes := []string{
		noteCreatedType, noteDeletedType,
		noteCreatedType, noteDeletedType,
		noteCreatedType, noteDeletedType,
	}
	gotTypes := make([]string, 0, len(records))

	for _, record := range records {
		require.Equal(t, note.ID.String(), string(record.Key))

		gotTypes = append(gotTypes, header(record, "ce_type"))
	}

	require.Equal(t, wantTypes, gotTypes)
}

func TestComponentForwarder_RestartLeavesNoGap(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := newEnv(t)
	repo := notes.New(env.pool, env.schemaIDs)
	observer := env.subscribe(t)

	createNotes := func(n int) {
		for range n {
			note := newNote("restart", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))
			observer.want(note.ID)
			require.NoError(t, repo.Create(t.Context(), author, note))
		}
	}

	createNotes(20)

	first := env.startForwarder(t)
	observer.next(t)
	require.NoError(t, first.Stop(t.Context()))

	createNotes(20)
	env.startForwarder(t)

	observer.waitAll(t)
}

func TestComponentForwarder_SendSpanIsARootLinkedToTheCreateSpan(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := newEnv(t)
	repo := notes.New(env.pool, env.schemaIDs)
	note := newNote("traced", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))

	require.NoError(t, repo.Create(t.Context(), author, note))
	env.startForwarder(t)

	messageID := env.subscribe(t, note.ID).next(t).ID
	create := endedSpan(t, "create notes.note.v1", messageID)
	send := endedSpan(t, "send notes.note.v1", messageID)

	require.False(t, send.Parent().IsValid(), "send must be a root span")
	require.NotEqual(t, create.SpanContext().TraceID(), send.SpanContext().TraceID())
	require.Equal(t, trace.SpanKindProducer, send.SpanKind())
	require.Len(t, send.Links(), 1)
	require.Equal(t, create.SpanContext().SpanID(), send.Links()[0].SpanContext.SpanID())
	require.Equal(t, create.SpanContext().TraceID(), send.Links()[0].SpanContext.TraceID())
	require.Equal(t, map[string]string{
		"messaging.system":            "kafka",
		"messaging.destination.name":  "notes.note.v1",
		"messaging.operation.name":    "send",
		"messaging.operation.type":    "send",
		"messaging.message.id":        messageID,
		"messaging.kafka.message.key": note.ID.String(),
		"messaging.delivery.attempt":  "1",
	}, attributesOf(send))
}

func TestComponentForwarder_RecordsSendAndProcessDurationsInSeconds(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := newEnv(t)
	repo := notes.New(env.pool, env.schemaIDs)
	note := newNote("timed", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))

	require.NoError(t, repo.Create(t.Context(), author, note))
	env.startForwarder(t)
	env.subscribe(t, note.ID).next(t)

	wantBounds := []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10}

	for _, name := range []string{"messaging.client.operation.duration", "messaging.process.duration"} {
		unit, bounds := histogram(t, name)
		require.Equal(t, "s", unit, name)
		require.Equal(t, wantBounds, bounds, name)
	}
}

func TestComponentForwarder_MessagingMetricsOmitTheDeliveryAttempt(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := newEnv(t)
	repo := notes.New(env.pool, env.schemaIDs)
	note := newNote("unbounded", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))

	require.NoError(t, repo.Create(t.Context(), author, note))
	env.startForwarder(t)
	env.subscribe(t, note.ID).next(t)

	names := []string{
		"messaging.client.sent.messages",
		"messaging.client.operation.duration",
		"messaging.process.duration",
	}
	recorded := messagingMetricAttributes(t, names...)

	for _, name := range names {
		for _, set := range recorded[name] {
			require.False(t, set.HasValue("messaging.delivery.attempt"), "%s carries %v", name, set.ToSlice())
		}
	}
}

func TestComponentForwarder_HoldsBackEverythingAfterAMessageWithoutAValidEnvelope(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := newEnv(t)
	repo := notes.New(env.pool, env.schemaIDs)
	note := newNote("after the malformed one", time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))

	malformed := message.NewMessage(uuid.NewString(), []byte("{}"))
	malformed.Metadata.Set("ce_subject", note.ID.String())
	malformed.Metadata.Set("ce_partitionkey", note.ID.String())

	tx, err := env.pool.Begin(t.Context())
	require.NoError(t, err)
	require.NoError(t, notes.Outbox.Publish(t.Context(), tx, events.Topic, malformed))
	require.NoError(t, tx.Commit(t.Context()))
	require.NoError(t, repo.Create(t.Context(), author, note))

	observer := env.subscribe(t, note.ID)
	env.startForwarder(t)

	select {
	case event := <-observer.events:
		t.Fatalf("the forwarder sent %s past a malformed message", event.Type)
	case <-time.After(5 * time.Second):
	}
}

func messagingMetricAttributes(t *testing.T, names ...string) map[string][]attribute.Set {
	t.Helper()

	var recorded map[string][]attribute.Set

	require.Eventually(t, func() bool {
		recorded = collectMetricAttributes(t)

		for _, name := range names {
			if len(recorded[name]) == 0 {
				return false
			}
		}

		return true
	}, waitFor, 50*time.Millisecond, "no data points for %v", names)

	return recorded
}

func collectMetricAttributes(t *testing.T) map[string][]attribute.Set {
	t.Helper()

	var collected metricdata.ResourceMetrics
	require.NoError(t, metrics.Collect(t.Context(), &collected))

	recorded := map[string][]attribute.Set{}

	for _, scope := range collected.ScopeMetrics {
		for _, m := range scope.Metrics {
			switch data := m.Data.(type) {
			case metricdata.Histogram[float64]:
				for _, point := range data.DataPoints {
					recorded[m.Name] = append(recorded[m.Name], point.Attributes)
				}
			case metricdata.Sum[int64]:
				for _, point := range data.DataPoints {
					recorded[m.Name] = append(recorded[m.Name], point.Attributes)
				}
			}
		}
	}

	return recorded
}

func histogram(t *testing.T, name string) (string, []float64) {
	t.Helper()

	messagingMetricAttributes(t, name)

	var collected metricdata.ResourceMetrics
	require.NoError(t, metrics.Collect(t.Context(), &collected))

	for _, scope := range collected.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != name {
				continue
			}

			data, ok := m.Data.(metricdata.Histogram[float64])
			require.True(t, ok, "%s is a %T", name, m.Data)
			require.NotEmpty(t, data.DataPoints, name)

			return m.Unit, data.DataPoints[0].Bounds
		}
	}

	t.Fatalf("no %s histogram", name)

	return "", nil
}

func endedSpan(t *testing.T, name, messageID string) sdktrace.ReadOnlySpan {
	t.Helper()

	var found sdktrace.ReadOnlySpan

	require.Eventually(t, func() bool {
		for _, span := range spans.Ended() {
			if span.Name() == name && attributesOf(span)["messaging.message.id"] == messageID {
				found = span

				return true
			}
		}

		return false
	}, waitFor, 50*time.Millisecond, "no ended %q span for message %s", name, messageID)

	return found
}

func attributesOf(span sdktrace.ReadOnlySpan) map[string]string {
	attributes := map[string]string{}
	for _, kv := range span.Attributes() {
		attributes[string(kv.Key)] = kv.Value.String()
	}

	return attributes
}

type env struct {
	*eventbustest.Env

	brokers   []string
	pool      *pgxpool.Pool
	schemaIDs eventbus.SchemaIDs
}

var createTopic sync.Once

func newEnv(t *testing.T) *env {
	t.Helper()

	e := eventbustest.New(t, events.Protos)

	createTopic.Do(func() { e.CreateTopic(t, events.Topic, 3) })

	schemaIDs, err := eventbus.LookupSchemaIDs(t.Context(), e.Registry(t), events.Protos)
	require.NoError(t, err)

	return &env{Env: e, brokers: e.Brokers, pool: testdb.New(t, migrations.FS), schemaIDs: schemaIDs}
}

func (e *env) startForwarder(t *testing.T) *outbox.Forwarder {
	t.Helper()

	f, err := forwarder.New(t.Context(), notes.Outbox, e.pool, e.brokers, func() { t.Errorf("forwarder stopped unexpectedly") })
	require.NoError(t, err)

	require.NoError(t, f.Start(t.Context()))
	require.NoError(t, f.Ready(t.Context()))

	t.Cleanup(func() { _ = f.Stop(context.Background()) })

	return f
}

func (e *env) subscribe(t *testing.T, ids ...domain.NoteID) *recorder {
	t.Helper()

	e.CreateTopic(t, eventbus.DLQTopic(events.Topic, e.Group), 1)

	r := &recorder{events: make(chan *eventbus.Event, 1000), wanted: map[string]bool{}, seen: map[string]bool{}}
	r.want(ids...)

	subscriber, err := eventbus.NewSubscriber(&eventbus.SubscriberConfig{
		Brokers: e.brokers,
		Group:   e.Group,
		Topics:  []string{events.Topic},
		Handler: r,
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

	return r
}

type recorder struct {
	mu     sync.Mutex
	wanted map[string]bool
	seen   map[string]bool
	events chan *eventbus.Event
}

func (r *recorder) Handle(_ context.Context, event *eventbus.Event) error {
	r.mu.Lock()

	wanted := r.wanted[event.Subject]
	if wanted {
		r.seen[event.Subject] = true
	}
	r.mu.Unlock()

	if !wanted {
		return nil
	}

	r.events <- event

	return nil
}

func (r *recorder) want(ids ...domain.NoteID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, id := range ids {
		r.wanted[id.String()] = true
	}
}

func (r *recorder) next(t *testing.T) *eventbus.Event {
	t.Helper()

	select {
	case event := <-r.events:
		return event
	case <-time.After(waitFor):
		t.Fatal("no event within the timeout")

		return nil
	}
}

func (r *recorder) waitAll(t *testing.T) {
	t.Helper()

	deadline := time.After(waitFor)

	for {
		r.mu.Lock()
		missing := 0

		for id := range r.wanted {
			if !r.seen[id] {
				missing++
			}
		}
		r.mu.Unlock()

		if missing == 0 {
			return
		}

		select {
		case <-r.events:
		case <-deadline:
			t.Fatalf("%d notes never reached Kafka", missing)
		}
	}
}

func consume(t *testing.T, brokers []string, id domain.NoteID, n int) []*sarama.ConsumerMessage {
	t.Helper()

	consumer, err := sarama.NewConsumer(brokers, sarama.NewConfig())
	require.NoError(t, err)

	t.Cleanup(func() { _ = consumer.Close() })

	partitions, err := consumer.Partitions(events.Topic)
	require.NoError(t, err)

	records := make(chan *sarama.ConsumerMessage, 1000)

	for _, partition := range partitions {
		pc, err := consumer.ConsumePartition(events.Topic, partition, sarama.OffsetOldest)
		require.NoError(t, err)

		t.Cleanup(func() { _ = pc.Close() })

		go func() {
			for record := range pc.Messages() {
				if header(record, "ce_subject") == id.String() {
					records <- record
				}
			}
		}()
	}

	got := make([]*sarama.ConsumerMessage, 0, n)

	for len(got) < n {
		select {
		case record := <-records:
			got = append(got, record)
		case <-time.After(waitFor):
			t.Fatalf("got %d of %d records of %s", len(got), n, id)
		}
	}

	return got
}

func header(record *sarama.ConsumerMessage, key string) string {
	for _, h := range record.Headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}

	return ""
}

func newNote(title string, createdAt time.Time) *domain.Note {
	return &domain.Note{
		ID:             domain.NewNoteID(),
		OrganizationID: author.OrganizationID,
		NoteContent:    domain.NoteContent{Title: title, Body: "body of " + title},
		CreatedAt:      createdAt,
	}
}

func mustParseUserID(s string) iamdomain.UserID {
	id, err := iamdomain.ParseUserID(s)
	if err != nil {
		panic(err)
	}

	return id
}
