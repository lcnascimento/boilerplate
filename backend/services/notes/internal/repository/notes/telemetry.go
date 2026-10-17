package notes

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

const pkg = "notes/repository/notes"

var tracer = otel.Tracer(pkg)

// onEmitStart opens the PRODUCER span and injects its context into msg, which carries it through
// the outbox and Kafka to the forwarder and consumer spans that link back to it.
func onEmitStart(ctx context.Context, msg *message.Message) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, "create "+events.Topic,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.MessagingSystemKafka,
			semconv.MessagingOperationTypeCreate,
			semconv.MessagingOperationName("create"),
			semconv.MessagingDestinationName(events.Topic),
			semconv.MessagingMessageID(msg.UUID),
			semconv.MessagingKafkaMessageKey(msg.Metadata.Get("ce_partitionkey")),
		),
	)

	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(msg.Metadata))

	return ctx, span
}

func onEmitEnd(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	span.End()
}
