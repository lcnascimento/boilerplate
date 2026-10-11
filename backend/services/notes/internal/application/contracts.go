package application

import (
	"context"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
)

type NotesRepository interface {
	Create(ctx context.Context, note *domain.Note) error
	Get(ctx context.Context, id domain.NoteID) (*domain.Note, error)
	List(ctx context.Context) ([]*domain.Note, error)
	Delete(ctx context.Context, id domain.NoteID) error
}

// Validator checks a request against its validate tags.
type Validator interface {
	Struct(s any) error
}
