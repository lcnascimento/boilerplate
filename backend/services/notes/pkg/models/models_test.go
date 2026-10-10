package models_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/validator"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/models"
)

// These tests run serially: go-kit validator v1.10.1 writes into the shared
// errors.ErrInvalidInput attribute map on every failed validation, which races.

// The title limit lives in a validate tag and in domain.NoteTitle; this test fails when they drift.
func TestCreateNoteRequest_TitleTagMatchesDomainRule(t *testing.T) {
	v, err := validator.New(validator.WithJSONFieldNames())
	require.NoError(t, err)

	for _, length := range []int{0, 1, domain.NoteTitleMaxLength, domain.NoteTitleMaxLength + 1} {
		title := strings.Repeat("é", length)

		tagErr := v.Struct(&models.CreateNoteRequest{Title: title})
		domainErr := domain.NoteTitle(title).Validate()

		require.Equal(t, domainErr == nil, tagErr == nil, "title with %d characters", length)
	}
}

func TestCreateNoteRequest_ValidationReportsEveryReason(t *testing.T) {
	v, err := validator.New(validator.WithJSONFieldNames())
	require.NoError(t, err)

	err = v.Struct(&models.CreateNoteRequest{Title: "", Body: strings.Repeat("a", 10_001)})

	require.ErrorIs(t, err, errors.ErrInvalidInput)
	require.Equal(t, []string{
		"[title] is a required field",
		"[body] must be a maximum of 10,000 characters in length",
	}, errors.SafeReasons(err))
}

func TestCreateNoteRequest_AcceptsMaximumBody(t *testing.T) {
	v, err := validator.New(validator.WithJSONFieldNames())
	require.NoError(t, err)

	require.NoError(t, v.Struct(&models.CreateNoteRequest{Title: "a", Body: strings.Repeat("a", 10_000)}))
}
