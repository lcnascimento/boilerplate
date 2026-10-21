package notes

import (
	"context"
	"time"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/roles"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/application"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/models"
)

const timestamptzPrecision = time.Microsecond

type Application struct {
	notes     application.NotesRepository
	validator application.Validator
	now       func() time.Time
}

func New(notes application.NotesRepository, v application.Validator, opts ...Option) *Application {
	app := &Application{
		notes:     notes,
		validator: v,
		now:       func() time.Time { return time.Now().UTC().Truncate(timestamptzPrecision) },
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

func (a *Application) Create(ctx context.Context, caller *actor.Actor, req *models.CreateNoteRequest) (*domain.Note, error) {
	ctx, span := a.onCreateStart(ctx)
	defer span.End()

	if err := caller.Authorize(ctx, roles.NoteCreate); err != nil {
		return nil, a.onError(ctx, err)
	}

	if req == nil {
		return nil, a.onError(ctx, domain.ErrInvalidInput.WithCause(errors.New("request body is required")))
	}

	note := &domain.Note{
		ID:             domain.NewNoteID(),
		OrganizationID: caller.OrganizationID,
		NoteContent:    req.NoteContent,
		CreatedAt:      a.now(),
	}

	if err := a.validate(note); err != nil {
		return nil, a.onError(ctx, err)
	}

	if err := a.notes.Create(ctx, note); err != nil {
		return nil, a.onError(ctx, ErrCreateNote.WithCause(err))
	}

	a.onCreated(ctx, note)

	return note, nil
}

func (a *Application) List(ctx context.Context, caller *actor.Actor) ([]*domain.Note, error) {
	ctx, span := a.onListStart(ctx)
	defer span.End()

	if err := caller.Authorize(ctx, roles.NoteRead); err != nil {
		return []*domain.Note{}, a.onError(ctx, err)
	}

	notes, err := a.notes.List(ctx, caller.OrganizationID)
	if err != nil {
		return []*domain.Note{}, a.onError(ctx, ErrListNotes.WithCause(err))
	}

	return notes, nil
}

func (a *Application) Get(ctx context.Context, caller *actor.Actor, id domain.NoteID) (*domain.Note, error) {
	ctx, span := a.onGetStart(ctx, id)
	defer span.End()

	if err := caller.Authorize(ctx, roles.NoteRead); err != nil {
		return nil, a.onError(ctx, err)
	}

	note, err := a.notes.Get(ctx, caller.OrganizationID, id)
	if err != nil {
		return nil, a.onError(ctx, ErrGetNote.WithCause(err))
	}

	return note, nil
}

func (a *Application) Delete(ctx context.Context, caller *actor.Actor, id domain.NoteID) error {
	ctx, span := a.onDeleteStart(ctx, id)
	defer span.End()

	if err := caller.Authorize(ctx, roles.NoteDelete); err != nil {
		return a.onError(ctx, err)
	}

	if err := a.notes.Delete(ctx, caller.OrganizationID, id); err != nil {
		return a.onError(ctx, ErrDeleteNote.WithCause(err))
	}

	a.onDeleted(ctx)

	return nil
}

func (a *Application) validate(note *domain.Note) error {
	err := a.validator.Struct(note)
	if err == nil {
		return nil
	}

	if !errors.Is(err, errors.ErrInvalidInput) {
		return err
	}

	out := domain.ErrInvalidInput
	for _, reason := range errors.SafeReasons(err) {
		out = out.WithCause(errors.New("%s", reason))
	}

	return out
}
