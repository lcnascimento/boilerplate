package models

import "github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"

type CreateNoteRequest struct {
	domain.NoteContent
}

type ListNotesResponse struct {
	Notes []*domain.Note `json:"notes"`
}
