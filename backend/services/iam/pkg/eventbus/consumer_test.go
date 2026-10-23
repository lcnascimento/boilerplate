package eventbus_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
)

func TestConsumer_ReadyFailsBeforeStart(t *testing.T) {
	t.Parallel()

	c := eventbus.NewConsumer(&eventbus.SubscriberConfig{Brokers: []string{"localhost:9092"}, Group: "audit"})

	require.ErrorIs(t, c.Ready(t.Context()), eventbus.ErrNotConsuming)
}
