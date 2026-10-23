package events_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/sr"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus/eventbustest"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"
)

var subjects = []string{
	"boilerplate.iam.v1.InvitationAccepted",
	"boilerplate.iam.v1.InvitationCanceled",
	"boilerplate.iam.v1.InvitationCreated",
	"boilerplate.iam.v1.InvitationDeclined",
	"boilerplate.iam.v1.MemberAdded",
	"boilerplate.iam.v1.MemberRemoved",
	"boilerplate.iam.v1.MemberRoleChanged",
	"boilerplate.iam.v1.OrganizationCreated",
	"boilerplate.iam.v1.OrganizationDeleted",
	"boilerplate.iam.v1.OrganizationRenamed",
	"boilerplate.iam.v1.PasswordChanged",
	"boilerplate.iam.v1.UserCreated",
	"boilerplate.iam.v1.UserDeleted",
}

func TestSubjects_OnePerProtoFile(t *testing.T) {
	t.Parallel()

	got, err := eventbus.Subjects(events.Protos)

	require.NoError(t, err)
	require.ElementsMatch(t, subjects, got)
}

func TestComponentRegistry_RegistersTheSchemasUnderFullTransitive(t *testing.T) {
	if testing.Short() {
		t.Skip("component test")
	}

	t.Parallel()

	client := eventbustest.New(t).Registry(t)

	ids, err := eventbus.RegisterSchemas(t.Context(), client, events.Protos)
	require.NoError(t, err)
	require.Len(t, ids, len(subjects))

	for _, result := range client.Compatibility(t.Context(), subjects...) {
		require.NoError(t, result.Err)
		require.Equal(t, sr.CompatFullTransitive, result.Level, result.Subject)
	}
}
