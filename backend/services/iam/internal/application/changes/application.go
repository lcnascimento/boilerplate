package changes

import (
	"context"
	"time"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/application"
)

const batchSize = 100

type Application struct {
	repo application.ChangesRepository
}

func New(repo application.ChangesRepository) *Application {
	return &Application{repo: repo}
}

func (a *Application) Translate(ctx context.Context) (int, error) {
	ctx, span := a.onTranslateStart(ctx)
	defer span.End()

	translated, err := a.repo.Translate(ctx, batchSize)
	if err != nil {
		return 0, a.onError(ctx, ErrTranslateChanges.WithCause(err))
	}

	a.onTranslated(ctx, translated)

	return len(translated), nil
}

func (a *Application) OldestPendingAge(ctx context.Context) (time.Duration, error) {
	age, err := a.repo.OldestPendingAge(ctx)
	if err != nil {
		return 0, ErrOldestPendingAge.WithCause(err)
	}

	return age, nil
}
