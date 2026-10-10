package models

import "github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"

type CreateNoteRequest struct {
	Title string `json:"title" validate:"required,max=120"`
	Body  string `json:"body"  validate:"max=10000"`
}

type ListNotesResponse struct {
	Notes []*domain.Note `json:"notes"`
}
