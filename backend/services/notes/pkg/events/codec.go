package events

import (
	"slices"

	"github.com/twmb/franz-go/pkg/sr"
	"google.golang.org/protobuf/proto"
)

// firstMessage is the Confluent message index of the only message in each .proto file.
var firstMessage = []int{0}

type SchemaIDs map[string]int

func (ids SchemaIDs) Encode(m proto.Message) ([]byte, error) {
	subject := string(m.ProtoReflect().Descriptor().FullName())

	id, ok := ids[subject]
	if !ok {
		return nil, ErrUnknownSchema.WithAttribute("subject", subject)
	}

	var header sr.ConfluentHeader

	value, err := header.AppendEncode(nil, id, firstMessage)
	if err != nil {
		return nil, ErrInvalidFraming.WithCause(err)
	}

	value, err = proto.MarshalOptions{}.MarshalAppend(value, m)
	if err != nil {
		return nil, ErrInvalidPayload.WithAttribute("subject", subject).WithCause(err)
	}

	return value, nil
}

func Decode(eventType string, value []byte) (schemaID int, message proto.Message, err error) {
	var header sr.ConfluentHeader

	schemaID, rest, err := header.DecodeID(value)
	if err != nil {
		return 0, nil, ErrInvalidFraming.WithCause(err)
	}

	index, rest, err := header.DecodeIndex(rest, len(firstMessage))
	if err != nil {
		return 0, nil, ErrInvalidFraming.WithCause(err)
	}

	if !slices.Equal(index, firstMessage) {
		return 0, nil, ErrInvalidFraming.WithAttribute("subject", eventType)
	}

	schemas, err := loadSchemas()
	if err != nil {
		return 0, nil, err
	}

	i := slices.IndexFunc(schemas, func(s schema) bool { return s.subject == eventType })
	if i < 0 {
		return schemaID, nil, nil
	}

	message = schemas[i].message.New().Interface()
	if err := proto.Unmarshal(rest, message); err != nil {
		return 0, nil, ErrInvalidPayload.WithAttribute("subject", eventType).WithCause(err)
	}

	return schemaID, message, nil
}
