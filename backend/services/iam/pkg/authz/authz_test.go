package authz_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/authz"
)

func TestParseRole_AcceptsKnownRoles(t *testing.T) {
	t.Parallel()

	inputs := map[string]authz.Role{
		"owner":  authz.RoleOwner,
		"admin":  authz.RoleAdmin,
		"viewer": authz.RoleViewer,
	}

	for input, want := range inputs {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			got, err := authz.ParseRole(input)

			require.NoError(t, err)
			require.Equal(t, want, got)
			require.Equal(t, input, got.String())
		})
	}
}

func TestParseRole_RejectsUnknownRoles(t *testing.T) {
	t.Parallel()

	inputs := []string{"", "member", "Owner", " owner", "owner,admin", "superadmin"}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			_, err := authz.ParseRole(input)

			require.ErrorIs(t, err, authz.ErrUnknownRole)
		})
	}
}

func TestAll_ListsRolesFromMostToLeastPrivileged(t *testing.T) {
	t.Parallel()

	require.Equal(t, []authz.Role{authz.RoleOwner, authz.RoleAdmin, authz.RoleViewer}, authz.AllRoles())
}

func TestRole_GrantsOrganizationUpdateToTheOwnerAndAdmin(t *testing.T) {
	t.Parallel()

	require.True(t, authz.RoleOwner.Grants(authz.PermissionOrganizationUpdate))
	require.True(t, authz.RoleAdmin.Grants(authz.PermissionOrganizationUpdate))
	require.False(t, authz.RoleViewer.Grants(authz.PermissionOrganizationUpdate))
	require.Equal(t, "organization:update", authz.PermissionOrganizationUpdate.String())
}

func TestRole_GrantsOrganizationDeleteOnlyToTheOwner(t *testing.T) {
	t.Parallel()

	require.True(t, authz.RoleOwner.Grants(authz.PermissionOrganizationDelete))
	require.False(t, authz.RoleAdmin.Grants(authz.PermissionOrganizationDelete))
	require.False(t, authz.RoleViewer.Grants(authz.PermissionOrganizationDelete))
	require.Equal(t, "organization:delete", authz.PermissionOrganizationDelete.String())
}

func TestRole_GrantsOrganizationTransferOnlyToTheOwner(t *testing.T) {
	t.Parallel()

	require.True(t, authz.RoleOwner.Grants(authz.PermissionOrganizationTransfer))
	require.False(t, authz.RoleAdmin.Grants(authz.PermissionOrganizationTransfer))
	require.False(t, authz.RoleViewer.Grants(authz.PermissionOrganizationTransfer))
	require.Equal(t, "organization:transfer", authz.PermissionOrganizationTransfer.String())
}

func TestRole_GrantsInvitationsToTheOwnerAndTheAdmin(t *testing.T) {
	t.Parallel()

	for _, p := range []authz.Permission{authz.PermissionInvitationCreate, authz.PermissionInvitationCancel} {
		require.True(t, authz.RoleOwner.Grants(p), p.String())
		require.True(t, authz.RoleAdmin.Grants(p), p.String())
		require.False(t, authz.RoleViewer.Grants(p), p.String())
	}

	require.Equal(t, "invitation:create", authz.PermissionInvitationCreate.String())
	require.Equal(t, "invitation:cancel", authz.PermissionInvitationCancel.String())
}

func TestRole_ZeroValueGrantsNothing(t *testing.T) {
	t.Parallel()

	var zero authz.Role

	for _, p := range authz.Permissions() {
		require.False(t, zero.Grants(p), p.String())
	}

	require.Empty(t, zero.Permissions())
}
