package eventbustest_test

import (
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/eventbustest"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/internal/testevents"
)

func TestComponentEventbustest_PublishWritesTheRealEnvelope(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	env := eventbustest.New(t, testevents.Protos)
	sent := &testevents.ThingDeleted{Id: "thing_1"}

	envelope := env.Publish(t, "thing_1", sent)

	record := consumeOne(t, env)
	headers := map[string]string{}

	for _, h := range record.Headers {
		headers[string(h.Key)] = string(h.Value)
	}

	got, err := eventbus.ParseEnvelope(headers)
	require.NoError(t, err)
	require.Equal(t, "thing_1", string(record.Key))
	require.Equal(t, envelope.ID, got.ID)
	require.Equal(t, "eventbustest", got.Source)
	require.Equal(t, string(sent.ProtoReflect().Descriptor().FullName()), got.Type)
	require.Equal(t, "thing_1", got.Subject)
	require.Equal(t, "thing_1", got.PartitionKey)
	require.Equal(t, "1.0", got.SpecVersion)
	require.Equal(t, "application/protobuf", got.DataContentType)
	require.WithinDuration(t, time.Now(), got.Time, time.Minute)

	schemaID, message, err := eventbus.Decode(got.Type, record.Value)
	require.NoError(t, err)
	require.Positive(t, schemaID)
	require.True(t, proto.Equal(sent, message), "want %v, got %v", sent, message)
}

func consumeOne(t *testing.T, env *eventbustest.Env) *sarama.ConsumerMessage {
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
