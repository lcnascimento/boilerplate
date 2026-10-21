package actor_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor/actortest"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/roles"
)

// example:begin
func TestActor_CanFollowsTheRoleNotePermissions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		role       roles.Role
		permission roles.Permission
		can        bool
	}{
		{roles.Owner, roles.NoteRead, true},
		{roles.Owner, roles.NoteCreate, true},
		{roles.Owner, roles.NoteDelete, true},
		{roles.Admin, roles.NoteRead, true},
		{roles.Admin, roles.NoteCreate, true},
		{roles.Admin, roles.NoteDelete, false},
		{roles.Viewer, roles.NoteRead, true},
		{roles.Viewer, roles.NoteCreate, false},
		{roles.Viewer, roles.NoteDelete, false},
	}

	for _, tc := range cases {
		t.Run(tc.role.String()+" "+tc.permission.String(), func(t *testing.T) {
			t.Parallel()

			a := actor.Actor{Role: tc.role}

			require.Equal(t, tc.can, a.Can(tc.permission))
		})
	}
}

// example:end

func TestActor_CannotUseAPermissionNoRoleGrants(t *testing.T) {
	t.Parallel()

	owner := actor.Actor{Role: roles.Owner}

	require.False(t, owner.Can(roles.Permission{}))
}

func TestFromContext_FailsAsAnInternalErrorWithoutAnActor(t *testing.T) {
	t.Parallel()

	_, err := actor.FromContext(t.Context())

	require.ErrorIs(t, err, actor.ErrNoActor)
	require.Equal(t, errors.KindInternal, errors.Kind(err))
}

func TestFromContext_ReturnsTheActorThatActortestPutThere(t *testing.T) {
	t.Parallel()

	orgID, err := domain.ParseOrganizationID("org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f")
	require.NoError(t, err)

	got, err := actor.FromContext(actortest.Context(t.Context(), actortest.Viewer(orgID)))

	require.NoError(t, err)
	require.Equal(t, "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f", got.OrganizationID.String())
	require.Equal(t, roles.Viewer, got.Role)
	require.Regexp(t, `^user_[0-9a-f-]{36}$`, got.UserID.String())
}
