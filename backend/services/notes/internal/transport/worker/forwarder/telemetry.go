package forwarder

import (
	"context"
	"strconv"
	"time"

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

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

const operationSend = "send"

var (
	pkg    = "notes/transport/worker/forwarder"
	logger = log.MustNewLogger(pkg)
	meter  = otel.Meter(pkg)
	tracer = otel.Tracer(pkg)

	sentMessages = metric.MustIntCounter(meter, "messaging.client.sent.messages",
		"Number of attempts to send a message to Kafka")
	operationDuration = mustSecondsHistogram("messaging.client.operation.duration",
		"Duration of one attempt to send a message to Kafka")

	oldestPendingAge = mustOldestPendingAgeGauge()

	ErrOldestPendingAge = errors.New("failed to observe the age of the oldest pending outbox message").
				WithCode("OUTBOX_OLDEST_PENDING_AGE")

	ErrSend = errors.New("failed to send the message to Kafka: retrying").WithCode("FORWARDER_SEND")
)

// secondsBuckets are the OTel semconv buckets for messaging durations.
var secondsBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10}

// mustSecondsHistogram replaces metric.MustFloat64Histogram, which declares milliseconds.
func mustSecondsHistogram(name, description string) otelmetric.Float64Histogram {
	histogram, err := meter.Float64Histogram(name,
		otelmetric.WithDescription(description),
		otelmetric.WithUnit("s"),
		otelmetric.WithExplicitBucketBoundaries(secondsBuckets...))
	if err != nil {
		panic(err)
	}

	return histogram
}

func mustOldestPendingAgeGauge() otelmetric.Int64ObservableGauge {
	gauge, err := meter.Int64ObservableGauge("notes.outbox.oldest_pending.age",
		otelmetric.WithDescription("Age of the oldest outbox message not yet forwarded to Kafka, 0 when none"),
		otelmetric.WithUnit("s"))
	if err != nil {
		panic(err)
	}

	return gauge
}

func registerOldestPendingAge(read func(context.Context) (time.Duration, error)) (otelmetric.Registration, error) {
	return meter.RegisterCallback(func(ctx context.Context, observer otelmetric.Observer) error {
		age, err := read(ctx)
		if err != nil {
			logger.Error(ctx, ErrOldestPendingAge.WithCause(err))

			return nil
		}

		observer.ObserveInt64(oldestPendingAge, int64(age.Seconds()))

		return nil
	}, oldestPendingAge)
}

func onStarted(ctx context.Context) {
	logger.Info(ctx, "forwarder started", log.String("topic", events.Topic))
}

func onStoppedUnexpectedly(ctx context.Context, err error) {
	logger.Error(ctx, ErrNotRunning.WithCause(err))
}

func commonAttributes(topic string) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.MessagingSystemKafka,
		semconv.MessagingDestinationName(topic),
		semconv.MessagingOperationName(operationSend),
		semconv.MessagingOperationTypeSend,
	}
}

func deliveryAttempt(n int) attribute.KeyValue {
	return attribute.Int("messaging.delivery.attempt", n)
}

// onSendStart opens a root span linked to the span that wrote the message to the outbox.
func onSendStart(msg *message.Message, topic string, attempt int) func(error) {
	started := time.Now()

	attributes := append(commonAttributes(topic), deliveryAttempt(attempt))

	if envelope, err := events.ParseEnvelope(msg.Metadata); err == nil {
		attributes = append(attributes,
			semconv.MessagingMessageID(envelope.ID),
			semconv.MessagingKafkaMessageKey(envelope.PartitionKey),
		)
	}

	opts := []trace.SpanStartOption{
		trace.WithNewRoot(),
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(attributes...),
	}

	producer := otel.GetTextMapPropagator().Extract(context.Background(), propagation.MapCarrier(msg.Metadata))
	if link := trace.LinkFromContext(producer); link.SpanContext.IsValid() {
		opts = append(opts, trace.WithLinks(link))
	}

	ctx, span := tracer.Start(msg.Context(), operationSend+" "+topic, opts...)

	return func(err error) {
		defer span.End()

		metricAttributes := commonAttributes(topic)

		if err != nil {
			errType := semconv.ErrorTypeKey.String(errorType(err))
			metricAttributes = append(metricAttributes, errType)

			span.SetAttributes(errType)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logger.Error(ctx, ErrSend.WithCause(err).WithAttribute("messaging.delivery.attempt", strconv.Itoa(attempt)))
		}

		sentMessages.Add(ctx, 1, otelmetric.WithAttributes(metricAttributes...))
		operationDuration.Record(ctx, time.Since(started).Seconds(), otelmetric.WithAttributes(metricAttributes...))
	}
}

func errorType(err error) string {
	if code := errors.Code(err); code != errors.CodeUnknown {
		return string(code)
	}

	return semconv.ErrorTypeOther.Value.AsString()
}
