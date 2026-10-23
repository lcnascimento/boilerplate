package roles_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/roles"
)

func TestRole_GrantsNotePermissions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		role       roles.Role
		permission roles.Permission
		granted    bool
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

			require.Equal(t, tc.granted, tc.role.Grants(tc.permission))
		})
	}
}

func TestPermission_StringJoinsResourceAndAction(t *testing.T) {
	t.Parallel()

	require.Equal(t, "note:read", roles.NoteRead.String())
	require.Equal(t, "note", roles.NoteDelete.Resource())
	require.Equal(t, "delete", roles.NoteDelete.Action())
}

func TestPermissions_ListsNotePermissionsAfterTheCoreOnes(t *testing.T) {
	t.Parallel()

	require.Equal(t, []roles.Permission{
		roles.OrganizationUpdate, roles.OrganizationDelete, roles.OrganizationTransfer,
		roles.NoteRead, roles.NoteCreate, roles.NoteDelete,
	}, roles.Permissions())
	require.Equal(t, []roles.Permission{
		roles.OrganizationUpdate, roles.NoteRead, roles.NoteCreate,
	}, roles.Admin.Permissions())
	require.Equal(t, []roles.Permission{roles.NoteRead}, roles.Viewer.Permissions())
}
