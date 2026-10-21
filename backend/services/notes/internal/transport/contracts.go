package transport

import (
	"context"
	"time"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/models"
)

type NotesService interface {
	Create(ctx context.Context, a *actor.Actor, req *models.CreateNoteRequest) (*domain.Note, error)
	List(ctx context.Context, a *actor.Actor) ([]*domain.Note, error)
	Get(ctx context.Context, a *actor.Actor, id domain.NoteID) (*domain.Note, error)
	Delete(ctx context.Context, a *actor.Actor, id domain.NoteID) error
}

type OutboxService interface {
	Purge(ctx context.Context) error
	OldestPendingAge(ctx context.Context) (time.Duration, error)
}

type Database interface {
	Ping(ctx context.Context) error
}
