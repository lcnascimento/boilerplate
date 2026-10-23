package eventbustest

import (
	"context"
	"io/fs"
	"maps"
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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
)

const (
	Image      = "redpandadata/redpanda:v26.2.5"
	partitions = 3
	source     = "eventbustest"
)

type cluster struct {
	brokers     []string
	registryURL string
	registry    *sr.Client
}

// Concurrent first registrations of one schema race into several IDs.
var registering sync.Mutex

var startCluster = sync.OnceValues(func() (cluster, error) {
	ctx := context.Background()

	container, err := redpanda.Run(ctx, Image)
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

	return cluster{brokers: []string{broker}, registryURL: registryURL, registry: registry}, nil
})

type Env struct {
	Brokers     []string
	RegistryURL string
	Topic       string
	Group       string

	publisher message.Publisher
	schemaIDs eventbus.SchemaIDs
}

func New(t *testing.T, protos ...fs.FS) *Env {
	t.Helper()

	c, err := startCluster()
	require.NoError(t, err, "start Redpanda")

	env := &Env{
		Brokers:     c.brokers,
		RegistryURL: c.registryURL,
		Topic:       RandomName(),
		Group:       RandomName(),
		schemaIDs:   eventbus.SchemaIDs{},
	}

	registering.Lock()
	defer registering.Unlock()

	for _, p := range protos {
		ids, err := eventbus.RegisterSchemas(t.Context(), c.registry, p)
		require.NoError(t, err, "register schemas")

		maps.Copy(env.schemaIDs, ids)
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

func (e *Env) Publish(t *testing.T, key string, m proto.Message) eventbus.Envelope {
	t.Helper()

	return e.PublishTo(t, e.Topic, key, m)
}

func (e *Env) PublishTo(t *testing.T, topic, key string, m proto.Message) eventbus.Envelope {
	t.Helper()

	return e.publishEvent(t.Context(), t, topic, key, m)
}

func (e *Env) PublishFrom(ctx context.Context, t *testing.T, key string, m proto.Message) eventbus.Envelope {
	t.Helper()

	return e.publishEvent(ctx, t, e.Topic, key, m)
}

func (e *Env) PublishRaw(t *testing.T, messageID string, headers map[string]string, value []byte) {
	t.Helper()

	e.publish(t, e.Topic, messageID, headers, value)
}

func (e *Env) publishEvent(ctx context.Context, t *testing.T, topic, key string, m proto.Message) eventbus.Envelope {
	t.Helper()

	value, err := e.schemaIDs.Encode(m)
	require.NoError(t, err)

	envelope := eventbus.Envelope{
		ID:              uuid.Must(uuid.NewV7()).String(),
		Source:          source,
		Type:            string(m.ProtoReflect().Descriptor().FullName()),
		Time:            time.Now().UTC(),
		Subject:         key,
		PartitionKey:    key,
		SpecVersion:     eventbus.SpecVersion,
		DataContentType: eventbus.ContentType,
	}

	headers := envelope.Headers()
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(headers))

	e.publish(t, topic, envelope.ID, headers, value)

	return envelope
}

func (e *Env) publish(t *testing.T, topic, messageID string, headers map[string]string, value []byte) {
	t.Helper()

	msg := message.NewMessage(messageID, value)
	for k, v := range headers {
		msg.Metadata.Set(k, v)
	}

	require.NoError(t, e.getPublisher(t).Publish(topic, msg))
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
