package eventbus_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/sr"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/eventbustest"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/internal/testevents"
)

func TestComponentRegistry_RegisterTwiceReturnsSameIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	client := eventbustest.New(t, testevents.Protos).Registry(t)

	first, err := eventbus.RegisterSchemas(t.Context(), client, testevents.Protos)
	require.NoError(t, err)

	second, err := eventbus.RegisterSchemas(t.Context(), client, testevents.Protos)
	require.NoError(t, err)

	subjects, err := eventbus.Subjects(testevents.Protos)
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

	client := eventbustest.New(t, testevents.Protos).Registry(t)

	_, err := eventbus.RegisterSchemas(t.Context(), client, testevents.Protos)
	require.NoError(t, err)

	subjects, err := eventbus.Subjects(testevents.Protos)
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

	client := eventbustest.New(t, testevents.Protos).Registry(t)

	registered, err := eventbus.RegisterSchemas(t.Context(), client, testevents.Protos)
	require.NoError(t, err)

	got, err := eventbus.LookupSchemaIDs(t.Context(), client, testevents.Protos)

	require.NoError(t, err)
	require.Equal(t, registered, got)
}

func TestComponentRegistry_RejectsIncompatibleChange(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	client := eventbustest.New(t, testevents.Protos).Registry(t)

	_, err := eventbus.RegisterSchemas(t.Context(), client, testevents.Protos)
	require.NoError(t, err)

	incompatible, err := os.ReadFile("testdata/thing_created_title_int64.proto")
	require.NoError(t, err)

	_, err = client.CreateSchema(
		sr.WithParams(t.Context(), sr.Normalize),
		fullName(&testevents.ThingCreated{}),
		sr.Schema{Schema: string(incompatible), Type: sr.TypeProtobuf},
	)

	var responseErr *sr.ResponseError

	require.True(t, errors.As(err, &responseErr), "want a registry rejection, got %v", err)
	require.Equal(t, http.StatusConflict, responseErr.StatusCode)
}
