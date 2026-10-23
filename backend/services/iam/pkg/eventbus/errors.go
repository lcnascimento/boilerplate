package eventbus

import "github.com/lcnascimento/go-kit/errors"

var (
	ErrMissingHeader = errors.New("required CloudEvents header is missing").
				WithCode("MISSING_HEADER").
				WithKind(errors.KindInvalidInput)

	ErrInvalidHeader = errors.New("CloudEvents header is malformed").
				WithCode("INVALID_HEADER").
				WithKind(errors.KindInvalidInput)

	ErrInvalidFraming = errors.New("value is not in the Confluent wire format with a single message index").
				WithCode("INVALID_FRAMING").
				WithKind(errors.KindInvalidInput)

	ErrInvalidPayload = errors.New("payload is not a valid Protobuf message of its type").
				WithCode("INVALID_PAYLOAD").
				WithKind(errors.KindInvalidInput)

	ErrUnknownSchema = errors.New("no schema ID for this subject: register it with make schemas.register").
				WithCode("UNKNOWN_SCHEMA")

	ErrRegisterSchema = errors.New("failed to register the schema in the Schema Registry").
				WithCode("REGISTER_SCHEMA")

	ErrLookupSchema = errors.New("schema is not registered in the Schema Registry: run make schemas.register").
			WithCode("LOOKUP_SCHEMA")

	ErrLoadSchemas = errors.New("embedded .proto files do not match the generated Go code: run make proto.generate").
			WithCode("LOAD_SCHEMAS")

	ErrNotOneMessagePerFile = errors.New("each .proto file must declare exactly one message").
				WithCode("NOT_ONE_MESSAGE_PER_FILE")

	ErrPermanent = errors.New("message cannot be processed and goes to the DLQ")

	ErrHandlerPanicked = errors.New("observer panicked while processing the message").
				WithCode("HANDLER_PANICKED")

	ErrInvalidSubscriberConfig = errors.New("subscriber needs brokers, a consumer group, topics and a handler").
					WithCode("INVALID_SUBSCRIBER_CONFIG").
					WithKind(errors.KindInvalidInput)

	ErrTopicMissing = errors.New("topic does not exist: declare it with make kafka.topics").
			WithCode("TOPIC_MISSING")

	ErrKafkaUnavailable = errors.New("failed to connect to Kafka").
				WithCode("KAFKA_UNAVAILABLE")

	ErrSubscriberStopped = errors.New("subscriber stopped with an error").
				WithCode("SUBSCRIBER_STOPPED")
)
