package fixture_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/fixture"
)

func TestRandomNote_IsValid(t *testing.T) {
	t.Parallel()

	for range 100 {
		require.NoError(t, fixture.RandomNote().Validate())
	}
}

func TestRandomNote_IsDistinct(t *testing.T) {
	t.Parallel()

	require.NotEqual(t, fixture.RandomNote().ID, fixture.RandomNote().ID)
}
