package forwarder

import (
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
)

type tracedPublisher struct {
	message.Publisher

	mu       sync.Mutex
	lastID   string
	attempts int
}

func newTracedPublisher(publisher message.Publisher) *tracedPublisher {
	return &tracedPublisher{Publisher: publisher}
}

func (p *tracedPublisher) Publish(topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		end := onSendStart(msg, topic, p.attempt(msg.UUID))

		err := p.Publisher.Publish(topic, msg)
		end(err)

		if err != nil {
			return err
		}
	}

	return nil
}

func (p *tracedPublisher) attempt(messageID string) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	if messageID != p.lastID {
		p.lastID = messageID
		p.attempts = 0
	}

	p.attempts++

	return p.attempts
}
