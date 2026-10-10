package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
)

func TestNote_MarshalsToWireContract(t *testing.T) {
	t.Parallel()

	note := domain.Note{
		ID:        domain.MustParseNoteID("note_01928f4e-7b3a-7c2d-9e1f-2a3b4c5d6e7f"),
		Title:     "Lista de compras",
		Body:      "",
		CreatedAt: time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC),
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

func TestNote_ValidateChecksIdentityAndTitle(t *testing.T) {
	t.Parallel()

	valid := domain.Note{ID: domain.NewNoteID(), Title: "Lista de compras"}
	require.NoError(t, valid.Validate())

	noID := domain.Note{Title: "Lista de compras"}
	require.ErrorIs(t, noID.Validate(), domain.ErrInvalidNoteID)

	noTitle := domain.Note{ID: domain.NewNoteID()}
	require.ErrorIs(t, noTitle.Validate(), domain.ErrInvalidNoteTitle)
}
