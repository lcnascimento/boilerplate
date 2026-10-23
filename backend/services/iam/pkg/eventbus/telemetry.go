package eventbus

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
	pkg    = "iam/pkg/eventbus"
	logger = log.MustNewLogger(pkg)
	meter  = otel.Meter(pkg)
	tracer = otel.Tracer(pkg)

	consumedMessages = metric.MustIntCounter(meter, "messaging.client.consumed.messages",
		"Number of messages the consumer finished handling, acked or nacked")
	processDuration = metric.MustDurationHistogram(meter, "messaging.process.duration",
		"Duration of one processing attempt of a message")
	dlqSentMessages = metric.MustIntCounter(meter, "messaging.dlq.sent.messages",
		"Number of messages sent to the DLQ")
)

func deliveryAttempt(n int) attribute.KeyValue {
	return attribute.Int("messaging.delivery.attempt", n)
}

func (s *Subscriber) commonAttributes(topic string) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.MessagingSystemKafka,
		semconv.MessagingDestinationName(topic),
		semconv.MessagingConsumerGroupName(s.group),
		semconv.MessagingOperationName(operationProcess),
		semconv.MessagingOperationTypeProcess,
	}
}

func (s *Subscriber) onAttemptStart(topic string, msg *message.Message, attempt int) (context.Context, func(error)) {
	ctx := msg.Context()
	started := time.Now()

	attributes := append(s.commonAttributes(topic), deliveryAttempt(attempt), semconv.MessagingMessageID(msg.Metadata.Get(headerID)))

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

	ctx, span := tracer.Start(ctx, operationProcess+" "+topic, opts...)

	return ctx, func(err error) {
		defer span.End()

		metricAttributes := s.commonAttributes(topic)

		if err != nil {
			errType := semconv.ErrorTypeKey.String(errorType(err))
			metricAttributes = append(metricAttributes, errType)

			span.SetAttributes(errType)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		processDuration.Record(ctx, time.Since(started).Seconds(), otelmetric.WithAttributes(metricAttributes...))
	}
}

func (s *Subscriber) onConsumed(topic string, msg *message.Message, err error) {
	attributes := s.commonAttributes(topic)
	if err != nil {
		attributes = append(attributes, semconv.ErrorTypeKey.String(errorType(err)))
	}

	consumedMessages.Add(msg.Context(), 1, otelmetric.WithAttributes(attributes...))
}

func (s *Subscriber) onDeadLettered(topic string, msg *message.Message) {
	ctx := msg.Context()
	errType := deliveryFrom(ctx).errorType

	attributes := append(s.commonAttributes(topic), semconv.ErrorTypeKey.String(errType))
	dlqSentMessages.Add(ctx, 1, otelmetric.WithAttributes(attributes...))

	logger.Warn(ctx, "message sent to the DLQ",
		log.String("dlq", DLQTopic(topic, s.group)), log.String("error.type", errType), log.String("message.id", msg.Metadata.Get(headerID)))
}

func onUndeliverable(ctx context.Context, err error) error {
	logger.ErrorBySeverity(ctx, err)

	return err
}

func onRouterStopped(ctx context.Context, err error) {
	logger.Error(ctx, err)
}

func errorType(err error) string {
	if code := errors.Code(err); code != errors.CodeUnknown {
		return string(code)
	}

	return semconv.ErrorTypeOther.Value.AsString()
}
