package changes

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lcnascimento/go-kit/o11y/log"
)

var (
	pkg    = "iam/application/changes"
	logger = log.MustNewLogger(pkg)
	tracer = otel.Tracer(pkg)
)

func (a *Application) onTranslateStart(ctx context.Context) (context.Context, trace.Span) {
	return tracer.Start(ctx, "TranslateIdentityChanges")
}

func (a *Application) onTranslated(ctx context.Context, translated int) {
	trace.SpanFromContext(ctx).SetAttributes(attribute.Int("iam.changes.translated", translated))
}

func (a *Application) onError(ctx context.Context, err error) error {
	logger.ErrorBySeverity(ctx, err)

	return err
}
