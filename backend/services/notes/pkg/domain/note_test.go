package domain_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/validator"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
)

func TestNote_MarshalsToWireContract(t *testing.T) {
	t.Parallel()

	note := domain.Note{
		ID:             domain.MustParseNoteID("note_01928f4e-7b3a-7c2d-9e1f-2a3b4c5d6e7f"),
		OrganizationID: iamdomain.MustParseOrganizationID("org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f"),
		NoteContent:    domain.NoteContent{Title: "Lista de compras", Body: ""},
		CreatedAt:      time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC),
	}

	out, err := json.Marshal(note)

	require.NoError(t, err)
	require.JSONEq(t, `{
		"id": "note_01928f4e-7b3a-7c2d-9e1f-2a3b4c5d6e7f",
		"title": "Lista de compras",
		"body": "",
		"created_at": "2026-10-10T19:30:00Z"
	}`, string(out))
}

func TestNote_ValidAcceptsBoundaryValues(t *testing.T) {
	t.Parallel()

	v := newValidator(t)

	valid := map[string]domain.NoteContent{
		"one character title":       {Title: "a"},
		"120 character title":       {Title: strings.Repeat("a", 120)},
		"120 multibyte runes title": {Title: strings.Repeat("ç", 120)},
		"10000 character body":      {Title: "a", Body: strings.Repeat("a", 10_000)},
	}

	for name, content := range valid {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, v.Struct(&domain.Note{ID: domain.NewNoteID(), NoteContent: content}))
		})
	}
}

func TestNote_InvalidReportsEveryReason(t *testing.T) {
	t.Parallel()

	v := newValidator(t)

	cases := map[string]struct {
		note    domain.Note
		reasons []string
	}{
		"zero id": {
			note:    domain.Note{NoteContent: domain.NoteContent{Title: "a"}},
			reasons: []string{"[id] must be a valid canonical id with prefix note"},
		},
		"empty title": {
			note:    domain.Note{ID: domain.NewNoteID()},
			reasons: []string{"[title] is a required field"},
		},
		"121 character title": {
			note:    domain.Note{ID: domain.NewNoteID(), NoteContent: domain.NoteContent{Title: strings.Repeat("ç", 121)}},
			reasons: []string{"[title] must be a maximum of 120 characters in length"},
		},
		"10001 character body": {
			note:    domain.Note{ID: domain.NewNoteID(), NoteContent: domain.NoteContent{Title: "a", Body: strings.Repeat("a", 10_001)}},
			reasons: []string{"[body] must be a maximum of 10,000 characters in length"},
		},
		"everything wrong": {
			note: domain.Note{NoteContent: domain.NoteContent{Body: strings.Repeat("a", 10_001)}},
			reasons: []string{
				"[title] is a required field",
				"[body] must be a maximum of 10,000 characters in length",
				"[id] must be a valid canonical id with prefix note",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := v.Struct(&tc.note)

			require.ErrorIs(t, err, errors.ErrInvalidInput)
			require.Equal(t, tc.reasons, errors.SafeReasons(err))
		})
	}
}

func newValidator(t *testing.T) *validator.Validator {
	t.Helper()

	v, err := validator.New(validator.WithJSONFieldNames())
	require.NoError(t, err)

	return v
}
