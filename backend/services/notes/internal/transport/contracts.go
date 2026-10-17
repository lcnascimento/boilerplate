package transport

import (
	"context"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/models"
)

type NotesService interface {
	Create(ctx context.Context, req *models.CreateNoteRequest) (*domain.Note, error)
	List(ctx context.Context) ([]*domain.Note, error)
	Get(ctx context.Context, id domain.NoteID) (*domain.Note, error)
	Delete(ctx context.Context, id domain.NoteID) error
}

type Database interface {
	Ping(ctx context.Context) error
}
