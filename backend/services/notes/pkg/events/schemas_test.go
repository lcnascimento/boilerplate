package events_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

func TestSubjects_OnePerProtoFile(t *testing.T) {
	t.Parallel()

	got, err := eventbus.Subjects(events.Protos)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{"boilerplate.notes.v1.NoteCreated", "boilerplate.notes.v1.NoteDeleted"}, got)
}
