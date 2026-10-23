package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad_GivesOwnerAndAdminTheirBetterAuthMembershipDefaults(t *testing.T) {
	t.Parallel()

	m, err := load()
	require.NoError(t, err)

	defaults := map[string]string{}
	for _, r := range m.roles {
		defaults[r.name] = r.betterAuthDefaults
	}

	require.Equal(t, map[string]string{"owner": "ownerAc", "admin": "adminAc", "viewer": ""}, defaults)
}
