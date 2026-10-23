package changes

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/lcnascimento/go-kit/o11y/log"
	"github.com/lcnascimento/go-kit/o11y/metric"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/idomain"
)

var (
	pkg    = "iam/application/changes"
	logger = log.MustNewLogger(pkg)
	meter  = otel.Meter(pkg)
	tracer = otel.Tracer(pkg)

	changesTranslated = metric.MustIntCounter(meter, "iam.identity_change.translated",
		"Number of identity changes translated into Integration Events, by event type")
)

func (a *Application) onTranslateStart(ctx context.Context) (context.Context, trace.Span) {
	return tracer.Start(ctx, "TranslateIdentityChanges")
}

func (a *Application) onTranslated(ctx context.Context, translated []*idomain.Event) {
	trace.SpanFromContext(ctx).SetAttributes(attribute.Int("iam.changes.translated", len(translated)))

	for _, event := range translated {
		changesTranslated.Add(ctx, 1, otelmetric.WithAttributes(attribute.String("cloudevents.event_type", event.Type())))
	}
}

func (a *Application) onError(ctx context.Context, err error) error {
	logger.ErrorBySeverity(ctx, err)

	return err
}
