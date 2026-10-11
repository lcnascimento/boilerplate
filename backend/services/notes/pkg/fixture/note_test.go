package fixture_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/validator"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/fixture"
)

func TestRandomNote_IsValid(t *testing.T) {
	t.Parallel()

	v, err := validator.New(validator.WithJSONFieldNames())
	require.NoError(t, err)

	for range 100 {
		require.NoError(t, v.Struct(fixture.RandomNote()))
	}
}

func TestRandomNote_IsDistinct(t *testing.T) {
	t.Parallel()

	require.NotEqual(t, fixture.RandomNote().ID, fixture.RandomNote().ID)
}
