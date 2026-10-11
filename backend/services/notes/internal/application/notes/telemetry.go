package notes

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/lcnascimento/go-kit/o11y/baggage"
	"github.com/lcnascimento/go-kit/o11y/log"
	"github.com/lcnascimento/go-kit/o11y/metric"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/domain"
)

var (
	pkg    = "notes/application/notes"
	logger = log.MustNewLogger(pkg)
	meter  = otel.Meter(pkg)
	tracer = otel.Tracer(pkg)

	notesCreated = metric.MustIntCounter(meter, "notes.notes.created", "Number of notes created")
	notesDeleted = metric.MustIntCounter(meter, "notes.notes.deleted", "Number of notes deleted")
)

func (a *Application) onCreateStart(ctx context.Context) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, "CreateNote")

	logger.Debug(ctx, "creating note")

	return ctx, span
}

func (a *Application) onCreated(ctx context.Context, note *domain.Note) {
	ctx = baggage.ContextWithMembers(ctx, baggage.NewMember(domain.ContextKeyNoteID, note.ID.String()))

	notesCreated.Add(ctx, 1)
	logger.Info(ctx, "note created")
}

func (a *Application) onListStart(ctx context.Context) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, "ListNotes")

	logger.Debug(ctx, "listing notes")

	return ctx, span
}

func (a *Application) onGetStart(ctx context.Context, id domain.NoteID) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, "GetNote")
	ctx = baggage.ContextWithMembers(ctx, baggage.NewMember(domain.ContextKeyNoteID, id.String()))

	logger.Debug(ctx, "getting note")

	return ctx, span
}

func (a *Application) onDeleteStart(ctx context.Context, id domain.NoteID) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, "DeleteNote")
	ctx = baggage.ContextWithMembers(ctx, baggage.NewMember(domain.ContextKeyNoteID, id.String()))

	logger.Debug(ctx, "deleting note")

	return ctx, span
}

func (a *Application) onDeleted(ctx context.Context) {
	notesDeleted.Add(ctx, 1)
	logger.Info(ctx, "note deleted")
}

func (a *Application) onError(ctx context.Context, err error) error {
	logger.ErrorBySeverity(ctx, err)

	return err
}
