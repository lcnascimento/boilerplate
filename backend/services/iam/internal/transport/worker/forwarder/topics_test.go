package forwarder_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"
)

func TestKafkaTopics_DeclaresEveryTopicTheForwarderPublishes(t *testing.T) {
	t.Parallel()

	makefile, err := os.ReadFile("../../../../Makefile")
	require.NoError(t, err)

	for _, topic := range events.Topics {
		require.Contains(t, string(makefile), "--topic "+topic+" ")
	}
}
