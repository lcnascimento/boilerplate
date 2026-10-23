package audit

import (
	"context"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/application"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/idomain"
)

type Application struct {
	repo      application.AuditEntriesRepository
	validator application.Validator
}

func New(repo application.AuditEntriesRepository, v application.Validator) *Application {
	return &Application{repo: repo, validator: v}
}

func (a *Application) Record(ctx context.Context, entry *idomain.Entry) error {
	ctx, span := a.onRecordStart(ctx, entry)
	defer span.End()

	if err := a.validate(entry); err != nil {
		return a.onError(ctx, eventbus.ErrPermanent.WithCause(err))
	}

	if err := a.repo.Insert(ctx, entry); err != nil {
		return a.onError(ctx, ErrRecordEntry.WithCause(err))
	}

	a.onRecorded(ctx, entry)

	return nil
}

func (a *Application) validate(entry *idomain.Entry) error {
	err := a.validator.Struct(entry)
	if err == nil {
		return nil
	}

	if !errors.Is(err, errors.ErrInvalidInput) {
		return err
	}

	out := ErrInvalidEnvelope
	for _, reason := range errors.SafeReasons(err) {
		out = out.WithCause(errors.New("%s", reason))
	}

	return out
}
