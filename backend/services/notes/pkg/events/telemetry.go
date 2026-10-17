package events

import (
	"context"
	"strconv"
	"time"

	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/o11y/log"
	"github.com/lcnascimento/go-kit/o11y/metric"
)

const operationProcess = "process"

var (
	pkg    = "notes/pkg/events"
	logger = log.MustNewLogger(pkg)
	meter  = otel.Meter(pkg)
	tracer = otel.Tracer(pkg)

	consumedMessages = metric.MustIntCounter(meter, "messaging.client.consumed.messages",
		"Number of messages the consumer finished handling, acked or nacked")
	processDuration = metric.MustFloat64Histogram(meter, "messaging.process.duration",
		"Duration of one processing attempt of a message, in seconds")
	dlqSentMessages = metric.MustIntCounter(meter, "messaging.dlq.sent.messages",
		"Number of messages sent to the DLQ")
)

func deliveryAttempt(n int) attribute.KeyValue {
	return attribute.Int("messaging.delivery.attempt", n)
}

func (s *Subscriber) commonAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.MessagingSystemKafka,
		semconv.MessagingDestinationName(s.topic),
		semconv.MessagingConsumerGroupName(s.group),
		semconv.MessagingOperationName(operationProcess),
		semconv.MessagingOperationTypeProcess,
	}
}

func (s *Subscriber) onAttemptStart(msg *message.Message, attempt int) (context.Context, func(error)) {
	ctx := msg.Context()
	started := time.Now()

	attributes := append(s.commonAttributes(), deliveryAttempt(attempt), semconv.MessagingMessageID(msg.Metadata.Get(headerID)))

	if key, ok := kafka.MessageKeyFromCtx(ctx); ok {
		attributes = append(attributes, semconv.MessagingKafkaMessageKey(string(key)))
	}

	if partition, ok := kafka.MessagePartitionFromCtx(ctx); ok {
		attributes = append(attributes, semconv.MessagingDestinationPartitionID(strconv.Itoa(int(partition))))
	}

	if offset, ok := kafka.MessagePartitionOffsetFromCtx(ctx); ok {
		attributes = append(attributes, semconv.MessagingKafkaOffset(int(offset)))
	}

	opts := []trace.SpanStartOption{
		trace.WithNewRoot(),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attributes...),
	}

	producer := otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(msg.Metadata))
	if link := trace.LinkFromContext(producer); link.SpanContext.IsValid() {
		opts = append(opts, trace.WithLinks(link))
	}

	ctx, span := tracer.Start(ctx, operationProcess+" "+s.topic, opts...)

	return ctx, func(err error) {
		defer span.End()

		metricAttributes := append(s.commonAttributes(), deliveryAttempt(attempt))

		if err != nil {
			errType := semconv.ErrorTypeKey.String(errorType(err))
			metricAttributes = append(metricAttributes, errType)

			span.SetAttributes(errType)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logger.ErrorBySeverity(ctx, err)
		}

		processDuration.Record(ctx, time.Since(started).Seconds(), otelmetric.WithAttributes(metricAttributes...))
	}
}

func (s *Subscriber) onConsumed(msg *message.Message, err error) {
	attributes := s.commonAttributes()
	if err != nil {
		attributes = append(attributes, semconv.ErrorTypeKey.String(errorType(err)))
	}

	consumedMessages.Add(msg.Context(), 1, otelmetric.WithAttributes(attributes...))
}

func (s *Subscriber) onDeadLettered(msg *message.Message) {
	ctx := msg.Context()
	errType := deliveryFrom(ctx).errorType

	attributes := append(s.commonAttributes(), semconv.ErrorTypeKey.String(errType))
	dlqSentMessages.Add(ctx, 1, otelmetric.WithAttributes(attributes...))

	logger.Warn(ctx, "message sent to the DLQ",
		log.String("dlq", s.dlqTopic()), log.String("error.type", errType), log.String("message.id", msg.Metadata.Get(headerID)))
}

func errorType(err error) string {
	if code := errors.Code(err); code != errors.CodeUnknown {
		return string(code)
	}

	return semconv.ErrorTypeOther.Value.AsString()
}
