package audit

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/lcnascimento/go-kit/o11y/baggage"
	"github.com/lcnascimento/go-kit/o11y/log"
	"github.com/lcnascimento/go-kit/o11y/metric"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/idomain"
)

var (
	pkg    = "audit/application/audit"
	logger = log.MustNewLogger(pkg)
	meter  = otel.Meter(pkg)
	tracer = otel.Tracer(pkg)

	entriesRecorded = metric.MustIntCounter(meter, "audit.entries.recorded",
		"Number of Audit Entries recorded, by event type, redeliveries included")
)

func (a *Application) onRecordStart(ctx context.Context, entry *idomain.Entry) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, "RecordAuditEntry")
	ctx = baggage.ContextWithMembers(ctx,
		baggage.NewMember("cloudevents.event_id", entry.ID),
		baggage.NewMember("cloudevents.event_source", entry.Source),
		baggage.NewMember("cloudevents.event_type", entry.Type),
	)

	logger.Debug(ctx, "recording audit entry")

	return ctx, span
}

func (a *Application) onRecorded(ctx context.Context, entry *idomain.Entry) {
	entriesRecorded.Add(ctx, 1, otelmetric.WithAttributes(attribute.String("cloudevents.event_type", entry.Type)))
	logger.Info(ctx, "audit entry recorded")
}

func (a *Application) onError(ctx context.Context, err error) error {
	logger.ErrorBySeverity(ctx, err)

	return err
}
