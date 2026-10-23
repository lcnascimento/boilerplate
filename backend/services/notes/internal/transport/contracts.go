package transport

import (
	"context"
	"net/http"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/models"
)

type NotesService interface {
	Create(ctx context.Context, caller *actor.Actor, req *models.CreateNoteRequest) (*domain.Note, error)
	List(ctx context.Context, caller *actor.Actor) ([]*domain.Note, error)
	Get(ctx context.Context, caller *actor.Actor, id domain.NoteID) (*domain.Note, error)
	Delete(ctx context.Context, caller *actor.Actor, id domain.NoteID) error
}

type Authenticator interface {
	Handler(next http.Handler) http.Handler
}

type Database interface {
	Ping(ctx context.Context) error
}
