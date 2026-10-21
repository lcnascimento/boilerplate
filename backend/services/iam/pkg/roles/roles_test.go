package roles_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/roles"
)

func TestParseRole_AcceptsKnownRoles(t *testing.T) {
	t.Parallel()

	inputs := map[string]roles.Role{
		"owner":  roles.Owner,
		"admin":  roles.Admin,
		"viewer": roles.Viewer,
	}

	for input, want := range inputs {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			got, err := roles.ParseRole(input)

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

			_, err := roles.ParseRole(input)

			require.ErrorIs(t, err, roles.ErrUnknownRole)
		})
	}
}

func TestAll_ListsRolesFromMostToLeastPrivileged(t *testing.T) {
	t.Parallel()

	require.Equal(t, []roles.Role{roles.Owner, roles.Admin, roles.Viewer}, roles.All())
}

func TestRole_ZeroValueGrantsNothing(t *testing.T) {
	t.Parallel()

	var zero roles.Role

	for _, p := range roles.Permissions() {
		require.False(t, zero.Grants(p), p.String())
	}

	require.Empty(t, zero.Permissions())
}
