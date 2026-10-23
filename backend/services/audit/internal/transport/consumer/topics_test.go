package consumer_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/transport/consumer"
)

func TestTopics_AreEveryIamTopic(t *testing.T) {
	t.Parallel()

	require.Equal(t, []string{
		"iam.user.v1",
		"iam.session.v1",
		"iam.organization.v1",
		"iam.invitation.v1",
		// example:begin
		"notes.note.v1",
		// example:end
	}, consumer.Topics)
}

func TestKafkaTopics_DeclaresTheDLQOfEveryTopic(t *testing.T) {
	t.Parallel()

	makefile, err := os.ReadFile("../../../Makefile")
	require.NoError(t, err)

	for _, topic := range consumer.Topics {
		require.Contains(t, string(makefile), "--topic "+eventbus.DLQTopic(topic, consumer.Group)+" ")
	}
}
