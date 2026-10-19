package eventstest

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redpanda"
	"github.com/twmb/franz-go/pkg/sr"
	"google.golang.org/protobuf/proto"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

const (
	image      = "redpandadata/redpanda:v26.2.5"
	partitions = 3
	source     = "eventstest"
)

type cluster struct {
	brokers     []string
	registryURL string
	schemaIDs   events.SchemaIDs
}

var startCluster = sync.OnceValues(func() (cluster, error) {
	ctx := context.Background()

	container, err := redpanda.Run(ctx, image)
	if err != nil {
		return cluster{}, err
	}

	broker, err := container.KafkaSeedBroker(ctx)
	if err != nil {
		return cluster{}, err
	}

	registryURL, err := container.SchemaRegistryAddress(ctx)
	if err != nil {
		return cluster{}, err
	}

	registry, err := sr.NewClient(sr.URLs(registryURL))
	if err != nil {
		return cluster{}, err
	}

	// Concurrent first registrations of one schema race into several IDs, so register before any test runs.
	schemaIDs, err := events.RegisterSchemas(ctx, registry)
	if err != nil {
		return cluster{}, err
	}

	return cluster{brokers: []string{broker}, registryURL: registryURL, schemaIDs: schemaIDs}, nil
})

type Env struct {
	Brokers     []string
	RegistryURL string
	Topic       string
	Group       string

	publisher message.Publisher
	schemaIDs events.SchemaIDs
}

func New(t *testing.T) *Env {
	t.Helper()

	c, err := startCluster()
	require.NoError(t, err, "start Redpanda")

	env := &Env{
		Brokers:     c.brokers,
		RegistryURL: c.registryURL,
		Topic:       RandomName(),
		Group:       RandomName(),
		schemaIDs:   c.schemaIDs,
	}

	env.CreateTopic(t, env.Topic, partitions)

	return env
}

func RandomName() string {
	return "test_" + uuid.NewString()
}

func (e *Env) CreateTopic(t *testing.T, topic string, partitions int32) {
	t.Helper()

	admin, err := sarama.NewClusterAdmin(e.Brokers, sarama.NewConfig())
	require.NoError(t, err)

	defer func() { _ = admin.Close() }()

	err = admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: partitions, ReplicationFactor: 1}, false)
	require.NoError(t, err)
}

func (e *Env) Registry(t *testing.T) *sr.Client {
	t.Helper()

	client, err := sr.NewClient(sr.URLs(e.RegistryURL))
	require.NoError(t, err)

	return client
}

func (e *Env) Publish(t *testing.T, key string, m proto.Message) events.Envelope {
	t.Helper()

	value, err := e.schemaIDs.Encode(m)
	require.NoError(t, err)

	envelope := events.Envelope{
		ID:              uuid.Must(uuid.NewV7()).String(),
		Source:          source,
		Type:            string(m.ProtoReflect().Descriptor().FullName()),
		Time:            time.Now().UTC(),
		Subject:         key,
		PartitionKey:    key,
		SpecVersion:     events.SpecVersion,
		DataContentType: events.ContentType,
	}

	e.PublishRaw(t, envelope.ID, envelope.Headers(), value)

	return envelope
}

func (e *Env) PublishRaw(t *testing.T, messageID string, headers map[string]string, value []byte) {
	t.Helper()

	msg := message.NewMessage(messageID, value)
	for k, v := range headers {
		msg.Metadata.Set(k, v)
	}

	require.NoError(t, e.getPublisher(t).Publish(e.Topic, msg))
}

func (e *Env) getPublisher(t *testing.T) message.Publisher {
	t.Helper()

	if e.publisher != nil {
		return e.publisher
	}

	publisher, err := kafka.NewPublisher(kafka.PublisherConfig{
		Brokers: e.Brokers,
		Marshaler: kafka.NewWithPartitioningMarshaler(func(_ string, msg *message.Message) (string, error) {
			return msg.Metadata.Get("ce_partitionkey"), nil
		}),
	}, watermill.NopLogger{})
	require.NoError(t, err)

	t.Cleanup(func() { _ = publisher.Close() })

	e.publisher = publisher

	return publisher
}
