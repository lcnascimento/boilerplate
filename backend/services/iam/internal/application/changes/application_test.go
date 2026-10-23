package changes_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/application/changes"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/application/internal/mocks"
)

func TestTranslate_TranslatesABatchOfOneHundred(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockChangesRepository(gomock.NewController(t))
	repo.EXPECT().Translate(gomock.Any(), int32(100)).Return(42, nil)

	translated, err := changes.New(repo).Translate(t.Context())

	require.NoError(t, err)
	require.Equal(t, 42, translated)
}

func TestTranslate_WrapsRepositoryErrors(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockChangesRepository(gomock.NewController(t))
	repo.EXPECT().Translate(gomock.Any(), gomock.Any()).Return(0, errors.ErrMock)

	_, err := changes.New(repo).Translate(t.Context())

	require.ErrorIs(t, err, changes.ErrTranslateChanges)
	require.ErrorIs(t, err, errors.ErrMock)
}
