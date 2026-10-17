package audit

import (
	"context"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/application"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/idomain"
)

type Application struct {
	repo application.AuditEntriesRepository
}

func New(repo application.AuditEntriesRepository) *Application {
	return &Application{repo: repo}
}

// Record fails with events.ErrPermanent when the envelope cannot identify the Entry; any other error is transient.
func (a *Application) Record(ctx context.Context, entry *idomain.Entry) error {
	ctx, span := a.onRecordStart(ctx, entry)
	defer span.End()

	if err := validateEnvelope(entry); err != nil {
		return a.onError(ctx, events.ErrPermanent.WithCause(err))
	}

	if err := a.repo.Insert(ctx, entry); err != nil {
		return a.onError(ctx, ErrRecordEntry.WithCause(err))
	}

	a.onRecorded(ctx)

	return nil
}

func validateEnvelope(entry *idomain.Entry) error {
	required := []struct {
		attribute string
		missing   bool
	}{
		{"id", entry.ID == ""},
		{"source", entry.Source == ""},
		{"type", entry.Type == ""},
		{"specversion", entry.SpecVersion == ""},
		{"time", entry.Time.IsZero()},
		{"datacontenttype", entry.DataContentType == ""},
	}

	for _, r := range required {
		if r.missing {
			return ErrInvalidEnvelope.WithAttribute("attribute", r.attribute)
		}
	}

	return nil
}
