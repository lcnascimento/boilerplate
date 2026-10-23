package events_test

import (
	"io/fs"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/sr"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events/eventstest"
)

func TestComponentRegistry_RegisterTwiceReturnsSameIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	client := eventstest.New(t).Registry(t)

	first, err := events.RegisterSchemas(t.Context(), client)
	require.NoError(t, err)

	second, err := events.RegisterSchemas(t.Context(), client)
	require.NoError(t, err)

	subjects, err := events.Subjects()
	require.NoError(t, err)

	require.Equal(t, first, second)
	require.Len(t, first, len(subjects))

	for _, subject := range subjects {
		require.Contains(t, first, subject)
	}
}

func TestComponentRegistry_SubjectsAreFullTransitive(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	client := eventstest.New(t).Registry(t)

	_, err := events.RegisterSchemas(t.Context(), client)
	require.NoError(t, err)

	subjects, err := events.Subjects()
	require.NoError(t, err)

	for _, result := range client.Compatibility(t.Context(), subjects...) {
		require.NoError(t, result.Err)
		require.Equal(t, sr.CompatFullTransitive, result.Level, result.Subject)
	}
}

func TestComponentRegistry_LookupFindsRegisteredIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	client := eventstest.New(t).Registry(t)

	registered, err := events.RegisterSchemas(t.Context(), client)
	require.NoError(t, err)

	got, err := events.LookupSchemaIDs(t.Context(), client)

	require.NoError(t, err)
	require.Equal(t, registered, got)
}

func TestComponentRegistry_RejectsIncompatibleChange(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	client := eventstest.New(t).Registry(t)

	_, err := events.RegisterSchemas(t.Context(), client)
	require.NoError(t, err)

	incompatible, err := os.ReadFile("testdata/note_created_title_int64.proto")
	require.NoError(t, err)

	_, err = client.CreateSchema(
		sr.WithParams(t.Context(), sr.Normalize),
		fullName(&events.NoteCreated{}),
		sr.Schema{Schema: string(incompatible), Type: sr.TypeProtobuf},
	)

	var responseErr *sr.ResponseError

	require.True(t, errors.As(err, &responseErr), "want a registry rejection, got %v", err)
	require.Equal(t, http.StatusConflict, responseErr.StatusCode)
}

func TestComponentRegistry_AcceptsTheSchemasOverTheirFirstVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	client := eventstest.New(t).Registry(t)

	testdata := os.DirFS("testdata/first")
	firstVersions := map[string]string{
		fullName(&events.NoteCreated{}): "note_created.proto",
		fullName(&events.NoteDeleted{}): "note_deleted.proto",
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

	_, err := events.RegisterSchemas(t.Context(), client)

	require.NoError(t, err)

	for subject := range firstVersions {
		versions, err := client.SubjectVersions(t.Context(), subject)
		require.NoError(t, err)
		require.Len(t, versions, 2, subject)
	}
}
