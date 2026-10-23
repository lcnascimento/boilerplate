package actor_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor/actortest"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/authz"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"
)

// example:begin
func TestActor_CanFollowsTheRoleNotePermissions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		role       authz.Role
		permission authz.Permission
		can        bool
	}{
		{authz.RoleOwner, authz.PermissionNoteRead, true},
		{authz.RoleOwner, authz.PermissionNoteCreate, true},
		{authz.RoleOwner, authz.PermissionNoteDelete, true},
		{authz.RoleAdmin, authz.PermissionNoteRead, true},
		{authz.RoleAdmin, authz.PermissionNoteCreate, true},
		{authz.RoleAdmin, authz.PermissionNoteDelete, false},
		{authz.RoleViewer, authz.PermissionNoteRead, true},
		{authz.RoleViewer, authz.PermissionNoteCreate, false},
		{authz.RoleViewer, authz.PermissionNoteDelete, false},
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

	owner := actor.Actor{Role: authz.RoleOwner}

	require.False(t, owner.Can(authz.Permission{}))
}

func TestActor_AuthorizeAnswersADeniedPermissionAsUnauthorized(t *testing.T) {
	t.Parallel()

	owner := actor.Actor{Role: authz.RoleOwner}

	err := owner.Authorize(t.Context(), authz.Permission{})

	require.ErrorIs(t, err, errors.ErrRequestUnauthorized)
	require.Equal(t, errors.KindUnauthorized, errors.Kind(err))
	require.Equal(t, errors.CodeType("ERR_UNAUTHORIZED"), errors.Code(err))
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
	require.Equal(t, authz.RoleViewer, got.Role)
	require.Regexp(t, `^user_[0-9a-f-]{36}$`, got.UserID.String())
}
