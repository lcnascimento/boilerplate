package notes

import (
	"context"
	"time"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/application"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/models"
)

type Application struct {
	notes     application.NotesRepository
	validator application.Validator
	now       func() time.Time
}

func New(notes application.NotesRepository, v application.Validator, opts ...Option) *Application {
	app := &Application{
		notes:     notes,
		validator: v,
		// Microseconds, the precision of timestamptz: Create returns what Get reads back.
		now: func() time.Time { return time.Now().UTC().Truncate(time.Microsecond) },
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

func (a *Application) Create(ctx context.Context, req *models.CreateNoteRequest) (*domain.Note, error) {
	ctx, span := a.onCreateStart(ctx)
	defer span.End()

	if req == nil {
		return nil, a.onError(ctx, domain.ErrInvalidInput.WithCause(errors.New("request body is required")))
	}

	note := &domain.Note{
		ID:          domain.NewNoteID(),
		NoteContent: req.NoteContent,
		CreatedAt:   a.now(),
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

func (a *Application) List(ctx context.Context) ([]*domain.Note, error) {
	ctx, span := a.onListStart(ctx)
	defer span.End()

	notes, err := a.notes.List(ctx)
	if err != nil {
		return []*domain.Note{}, a.onError(ctx, ErrListNotes.WithCause(err))
	}

	return notes, nil
}

func (a *Application) Get(ctx context.Context, id domain.NoteID) (*domain.Note, error) {
	ctx, span := a.onGetStart(ctx, id)
	defer span.End()

	note, err := a.notes.Get(ctx, id)
	if err != nil {
		return nil, a.onError(ctx, ErrGetNote.WithCause(err))
	}

	return note, nil
}

func (a *Application) Delete(ctx context.Context, id domain.NoteID) error {
	ctx, span := a.onDeleteStart(ctx, id)
	defer span.End()

	if err := a.notes.Delete(ctx, id); err != nil {
		return a.onError(ctx, ErrDeleteNote.WithCause(err))
	}

	a.onDeleted(ctx)

	return nil
}

// validate runs the validator and rebuilds its failure as domain.ErrInvalidInput
// (go-kit's sentinel has another code), one cause per reason.
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
