package eventbus_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/internal/testevents"
)

func TestSubjects_OnePerProtoFile(t *testing.T) {
	t.Parallel()

	got, err := eventbus.Subjects(testevents.Protos)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{"boilerplate.eventbustest.v1.ThingCreated", "boilerplate.eventbustest.v1.ThingDeleted"}, got)
}

func TestSubjects_FailsForAProtoFileTheGoCodeDidNotRegister(t *testing.T) {
	t.Parallel()

	protos := fstest.MapFS{"boilerplate/eventbustest/v1/thing_archived.proto": {Data: []byte(`syntax = "proto3";`)}}

	_, err := eventbus.Subjects(protos)

	require.ErrorIs(t, err, eventbus.ErrLoadSchemas)
}

func fullName(m proto.Message) string {
	return string(m.ProtoReflect().Descriptor().FullName())
}
