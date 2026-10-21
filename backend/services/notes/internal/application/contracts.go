package application

import (
	"context"
	"time"

	iamdomain "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/domain"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
)

type NotesRepository interface {
	Create(ctx context.Context, caller *actor.Actor, note *domain.Note) error
	Get(ctx context.Context, orgID iamdomain.OrganizationID, id domain.NoteID) (*domain.Note, error)
	List(ctx context.Context, orgID iamdomain.OrganizationID) ([]*domain.Note, error)
	Delete(ctx context.Context, caller *actor.Actor, id domain.NoteID) error
}

type OutboxRepository interface {
	Purge(ctx context.Context, retention time.Duration) (int64, error)
	OldestPendingAge(ctx context.Context) (time.Duration, error)
}

type Validator interface {
	Struct(s any) error
}
