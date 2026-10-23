package consumer_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/consumer"
)

func TestKafkaTopics_DeclaresTheDLQOfEveryTopic(t *testing.T) {
	t.Parallel()

	makefile, err := os.ReadFile("../../../Makefile")
	require.NoError(t, err)

	for _, topic := range consumer.Topics {
		require.Contains(t, string(makefile), "--topic "+eventbus.DLQTopic(topic, consumer.Group)+" ")
	}
}
