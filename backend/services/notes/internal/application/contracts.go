package application

import (
	"context"
	"time"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
)

type NotesRepository interface {
	Create(ctx context.Context, note *domain.Note) error
	Get(ctx context.Context, id domain.NoteID) (*domain.Note, error)
	List(ctx context.Context) ([]*domain.Note, error)
	Delete(ctx context.Context, id domain.NoteID) error
}

type OutboxRepository interface {
	Purge(ctx context.Context, retention time.Duration) (int64, error)
	OldestPendingAge(ctx context.Context) (time.Duration, error)
}

type Validator interface {
	Struct(s any) error
}
