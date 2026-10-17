package audit

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lcnascimento/go-kit/o11y/log"
	"github.com/lcnascimento/go-kit/o11y/metric"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/idomain"
)

var (
	pkg    = "audit/application/audit"
	logger = log.MustNewLogger(pkg)
	meter  = otel.Meter(pkg)
	tracer = otel.Tracer(pkg)

	entriesRecorded = metric.MustIntCounter(meter, "audit.entries.recorded", "Number of Audit Entries recorded, redeliveries included")
)

func (a *Application) onRecordStart(ctx context.Context, entry *idomain.Entry) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, "RecordAuditEntry", trace.WithAttributes(
		attribute.String("cloudevents.event_id", entry.ID),
		attribute.String("cloudevents.event_source", entry.Source),
		attribute.String("cloudevents.event_type", entry.Type),
	))

	logger.Debug(ctx, "recording audit entry")

	return ctx, span
}

func (a *Application) onRecorded(ctx context.Context) {
	entriesRecorded.Add(ctx, 1)
	logger.Info(ctx, "audit entry recorded")
}

func (a *Application) onError(ctx context.Context, err error) error {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	logger.ErrorBySeverity(ctx, err)

	return err
}
