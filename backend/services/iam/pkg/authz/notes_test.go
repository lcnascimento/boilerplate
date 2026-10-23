package authz_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/authz"
)

func TestRole_GrantsNotePermissions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		role       authz.Role
		permission authz.Permission
		granted    bool
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

			require.Equal(t, tc.granted, tc.role.Grants(tc.permission))
		})
	}
}

func TestPermission_StringJoinsResourceAndAction(t *testing.T) {
	t.Parallel()

	require.Equal(t, "note:read", authz.PermissionNoteRead.String())
	require.Equal(t, "note", authz.PermissionNoteDelete.Resource())
	require.Equal(t, "delete", authz.PermissionNoteDelete.Action())
}

func TestPermissions_ListsNotePermissionsAfterTheCoreOnes(t *testing.T) {
	t.Parallel()

	require.Equal(t, []authz.Permission{
		authz.PermissionOrganizationUpdate, authz.PermissionOrganizationDelete, authz.PermissionOrganizationTransfer,
		authz.PermissionInvitationCreate, authz.PermissionInvitationCancel,
		authz.PermissionNoteRead, authz.PermissionNoteCreate, authz.PermissionNoteDelete,
	}, authz.Permissions())
	require.Equal(t, []authz.Permission{
		authz.PermissionOrganizationUpdate, authz.PermissionInvitationCreate, authz.PermissionInvitationCancel, authz.PermissionNoteRead, authz.PermissionNoteCreate,
	}, authz.RoleAdmin.Permissions())
	require.Equal(t, []authz.Permission{authz.PermissionNoteRead}, authz.RoleViewer.Permissions())
}
