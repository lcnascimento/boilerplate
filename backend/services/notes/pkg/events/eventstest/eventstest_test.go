package eventstest_test

import (
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events/eventstest"
)

func TestComponentEventstest_PublishWritesTheRealEnvelope(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := eventstest.New(t)
	sent := &events.NoteDeleted{Id: "note_1"}

	envelope := env.Publish(t, "note_1", sent)

	record := consumeOne(t, env)
	headers := map[string]string{}

	for _, h := range record.Headers {
		headers[string(h.Key)] = string(h.Value)
	}

	got, err := events.ParseEnvelope(headers)
	require.NoError(t, err)
	require.Equal(t, "note_1", string(record.Key))
	require.Equal(t, envelope.ID, got.ID)
	require.Equal(t, "eventstest", got.Source)
	require.Equal(t, string(sent.ProtoReflect().Descriptor().FullName()), got.Type)
	require.Equal(t, "note_1", got.Subject)
	require.Equal(t, "note_1", got.PartitionKey)
	require.Equal(t, "1.0", got.SpecVersion)
	require.Equal(t, "application/protobuf", got.DataContentType)
	require.WithinDuration(t, time.Now(), got.Time, time.Minute)

	schemaID, message, err := events.Decode(got.Type, record.Value)
	require.NoError(t, err)
	require.Positive(t, schemaID)
	require.True(t, proto.Equal(sent, message), "want %v, got %v", sent, message)
}

func consumeOne(t *testing.T, env *eventstest.Env) *sarama.ConsumerMessage {
	t.Helper()

	consumer, err := sarama.NewConsumer(env.Brokers, sarama.NewConfig())
	require.NoError(t, err)

	t.Cleanup(func() { _ = consumer.Close() })

	partitions, err := consumer.Partitions(env.Topic)
	require.NoError(t, err)

	records := make(chan *sarama.ConsumerMessage)

	for _, partition := range partitions {
		pc, err := consumer.ConsumePartition(env.Topic, partition, sarama.OffsetOldest)
		require.NoError(t, err)

		t.Cleanup(func() { _ = pc.Close() })

		go func() {
			for record := range pc.Messages() {
				records <- record
			}
		}()
	}

	select {
	case record := <-records:
		return record
	case <-time.After(30 * time.Second):
		t.Fatal("no record published")

		return nil
	}
}
