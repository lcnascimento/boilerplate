package eventbus_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redpanda"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/eventbustest"
)

const healthCheckBudget = 2 * time.Second

func TestComponentCluster_PingAnswersWhileTheBrokerIsUp(t *testing.T) {
	if testing.Short() {
		t.Skip("needs a Kafka broker")
	}

	t.Parallel()

	env := eventbustest.New(t)

	cluster, err := eventbus.DialCluster(env.Brokers, env.Topic)
	require.NoError(t, err)

	t.Cleanup(func() { _ = cluster.Close() })

	require.NoError(t, cluster.Ping(t.Context()))
}

func TestComponentCluster_PingFailsWithinAHealthCheckOnceTheBrokerIsGone(t *testing.T) {
	if testing.Short() {
		t.Skip("needs a Kafka broker")
	}

	t.Parallel()

	container, err := redpanda.Run(t.Context(), eventbustest.Image)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err)

	broker, err := container.KafkaSeedBroker(t.Context())
	require.NoError(t, err)

	cluster, err := eventbus.DialCluster([]string{broker})
	require.NoError(t, err)

	t.Cleanup(func() { _ = cluster.Close() })

	require.NoError(t, cluster.Ping(t.Context()))
	require.NoError(t, container.Stop(t.Context(), nil))

	for range 3 {
		started := time.Now()

		require.ErrorIs(t, cluster.Ping(t.Context()), eventbus.ErrKafkaUnavailable)
		require.Less(t, time.Since(started), healthCheckBudget)
	}
}

func TestDialCluster_FailsWithoutABroker(t *testing.T) {
	t.Parallel()

	_, err := eventbus.DialCluster([]string{"127.0.0.1:1"})

	require.ErrorIs(t, err, eventbus.ErrKafkaUnavailable)
}
