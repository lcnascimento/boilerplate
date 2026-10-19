package outbox

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/lcnascimento/go-kit/o11y/log"
	"github.com/lcnascimento/go-kit/o11y/metric"
)

var (
	pkg    = "notes/application/outbox"
	logger = log.MustNewLogger(pkg)
	meter  = otel.Meter(pkg)
	tracer = otel.Tracer(pkg)

	purgedMessages = metric.MustIntCounter(meter, "notes.outbox.purged.messages",
		"Number of forwarded messages deleted from the outbox")
	purgeRuns = metric.MustIntCounter(meter, "notes.outbox.purge.runs",
		"Number of outbox purges, by outcome")

	outcomeSuccess = otelmetric.WithAttributes(attribute.String("outcome", "success"))
	outcomeFailure = otelmetric.WithAttributes(attribute.String("outcome", "failure"))
)

func (a *Application) onPurgeStart(ctx context.Context) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, "PurgeOutbox")

	logger.Debug(ctx, "purging outbox")

	return ctx, span
}

func (a *Application) onPurged(ctx context.Context, purged int64) {
	purgedMessages.Add(ctx, purged)
	purgeRuns.Add(ctx, 1, outcomeSuccess)
	logger.Info(ctx, "outbox purged")
}

func (a *Application) onPurgeFailed(ctx context.Context, purged int64, err error) error {
	purgedMessages.Add(ctx, purged)
	purgeRuns.Add(ctx, 1, outcomeFailure)

	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	logger.ErrorBySeverity(ctx, err)

	return err
}
