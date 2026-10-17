package outbox

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5"
)

const (
	// ForwarderTopic names the outbox tables: watermill_notes_outbox and watermill_offsets_notes_outbox.
	ForwarderTopic = "notes_outbox"
	ConsumerGroup  = "notes-forwarder"
)

// NewPublisher writes messages to the outbox inside tx, so they exist if and only if tx commits.
// Publish takes the Kafka topic the forwarder delivers them to.
func NewPublisher(tx pgx.Tx) (message.Publisher, error) {
	publisher, err := sql.NewPublisher(
		sql.TxFromPgx(tx),
		sql.PublisherConfig{SchemaAdapter: sql.DefaultPostgreSQLSchema{}},
		watermill.NopLogger{},
	)
	if err != nil {
		return nil, err
	}

	return forwarder.NewPublisher(publisher, forwarder.PublisherConfig{ForwarderTopic: ForwarderTopic}), nil
}
