package models_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/models"
)

func TestCreateNoteRequest_DecodesFlatWireContract(t *testing.T) {
	t.Parallel()

	var req models.CreateNoteRequest

	err := json.Unmarshal([]byte(`{"title":"Lista de compras","body":"Leite."}`), &req)

	require.NoError(t, err)
	require.Equal(t, models.CreateNoteRequest{
		NoteContent: domain.NoteContent{Title: "Lista de compras", Body: "Leite."},
	}, req)
}
