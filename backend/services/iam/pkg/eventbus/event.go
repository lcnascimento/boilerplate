package eventbus

import (
	"context"
	"maps"

	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/proto"
)

type Handler interface {
	Handle(ctx context.Context, event *Event) error
}

type Event struct {
	Envelope

	Headers   map[string]string
	Payload   []byte
	SchemaID  int
	Topic     string
	Partition int32
	Offset    int64
	// Message is nil when no Go type of the event's ce_type is linked into the process.
	Message proto.Message
}

func parseEvent(topic string, msg *message.Message) (Event, error) {
	headers := make(map[string]string, len(msg.Metadata)+1)
	maps.Copy(headers, msg.Metadata)

	if msg.UUID != "" {
		headers[kafka.UUIDHeaderKey] = msg.UUID
	}

	envelope, err := ParseEnvelope(headers)
	if err != nil {
		return Event{}, err
	}

	schemaID, decoded, err := Decode(envelope.Type, msg.Payload)
	if err != nil {
		return Event{}, err
	}

	partition, _ := kafka.MessagePartitionFromCtx(msg.Context())
	offset, _ := kafka.MessagePartitionOffsetFromCtx(msg.Context())

	return Event{
		Envelope:  envelope,
		Headers:   headers,
		Payload:   msg.Payload,
		SchemaID:  schemaID,
		Topic:     topic,
		Partition: partition,
		Offset:    offset,
		Message:   decoded,
	}, nil
}
