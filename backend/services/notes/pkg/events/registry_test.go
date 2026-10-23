package events_test

import (
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/sr"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/eventbustest"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

func TestComponentRegistry_AcceptsTheSchemasOverTheirFirstVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	client := eventbustest.New(t, events.Protos).Registry(t)

	testdata := os.DirFS("testdata/first")
	firstVersions := map[string]string{
		"boilerplate.notes.v1.NoteCreated": "note_created.proto",
		"boilerplate.notes.v1.NoteDeleted": "note_deleted.proto",
	}

	for subject, name := range firstVersions {
		text, err := fs.ReadFile(testdata, name)
		require.NoError(t, err)

		_, err = client.CreateSchema(
			sr.WithParams(t.Context(), sr.Normalize),
			subject,
			sr.Schema{Schema: string(text), Type: sr.TypeProtobuf},
		)
		require.NoError(t, err)
	}

	_, err := eventbus.RegisterSchemas(t.Context(), client, events.Protos)

	require.NoError(t, err)

	for subject := range firstVersions {
		versions, err := client.SubjectVersions(t.Context(), subject)
		require.NoError(t, err)
		require.Len(t, versions, 2, subject)
	}
}
