package events

import (
	"context"
	"maps"

	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/proto"
)

type Observer interface {
	OnNoteCreated(ctx context.Context, event *Event, note *NoteCreated) error
	OnNoteDeleted(ctx context.Context, event *Event, note *NoteDeleted) error
	OnAny(ctx context.Context, event *Event) error
}

type UnimplementedObserver struct{}

func (UnimplementedObserver) OnNoteCreated(context.Context, *Event, *NoteCreated) error { return nil }

func (UnimplementedObserver) OnNoteDeleted(context.Context, *Event, *NoteDeleted) error { return nil }

func (UnimplementedObserver) OnAny(context.Context, *Event) error { return nil }

type Event struct {
	Envelope

	Headers   map[string]string
	Payload   []byte
	SchemaID  int
	Topic     string
	Partition int32
	Offset    int64
	Message   proto.Message
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

func notify(ctx context.Context, observer Observer, event *Event) error {
	var err error

	switch m := event.Message.(type) {
	case *NoteCreated:
		err = observer.OnNoteCreated(ctx, event, m)
	case *NoteDeleted:
		err = observer.OnNoteDeleted(ctx, event, m)
	}

	if err != nil {
		return err
	}

	return observer.OnAny(ctx, event)
}
