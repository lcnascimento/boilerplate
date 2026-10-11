package domain_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
)

func TestNoteID_NewHasNotePrefix(t *testing.T) {
	t.Parallel()

	id := domain.NewNoteID()

	require.True(t, strings.HasPrefix(id.String(), "note_"), id.String())
}

func TestNoteID_ParseRoundTrips(t *testing.T) {
	t.Parallel()

	id, err := domain.ParseNoteID("note_01928f4e-7b3a-7c2d-9e1f-2a3b4c5d6e7f")

	require.NoError(t, err)
	require.Equal(t, "note_01928f4e-7b3a-7c2d-9e1f-2a3b4c5d6e7f", id.String())
}

func TestNoteID_ParseRejectsMalformedIDs(t *testing.T) {
	t.Parallel()

	inputs := map[string]string{
		"empty":        "",
		"wrong prefix": "sch_01928f4e-7b3a-7c2d-9e1f-2a3b4c5d6e7f",
		"no prefix":    "01928f4e-7b3a-7c2d-9e1f-2a3b4c5d6e7f",
		"bad uuid":     "note_not-a-uuid",
	}

	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.ParseNoteID(input)

			require.ErrorIs(t, err, domain.ErrInvalidNoteID)
		})
	}
}

func TestNoteID_MarshalsAsPrefixedString(t *testing.T) {
	t.Parallel()

	id := domain.MustParseNoteID("note_01928f4e-7b3a-7c2d-9e1f-2a3b4c5d6e7f")

	out, err := json.Marshal(map[string]domain.NoteID{"id": id})

	require.NoError(t, err)
	require.JSONEq(t, `{"id":"note_01928f4e-7b3a-7c2d-9e1f-2a3b4c5d6e7f"}`, string(out))
}

func TestNoteID_UnmarshalsFromPrefixedString(t *testing.T) {
	t.Parallel()

	var out struct {
		ID domain.NoteID `json:"id"`
	}

	err := json.Unmarshal([]byte(`{"id":"note_01928f4e-7b3a-7c2d-9e1f-2a3b4c5d6e7f"}`), &out)

	require.NoError(t, err)
	require.Equal(t, "note_01928f4e-7b3a-7c2d-9e1f-2a3b4c5d6e7f", out.ID.String())
}
