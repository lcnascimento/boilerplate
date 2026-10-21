package events

import (
	"context"
	"time"

	"github.com/IBM/sarama"
)

const PingTimeout = time.Second

type Cluster struct {
	client sarama.Client
	topics []string
}

// DialCluster connects to the brokers; Ping then asks only for the metadata of the topics the process uses, since a
// leaderless partition elsewhere (__consumer_offsets while the broker starts) does not stop it.
func DialCluster(brokers []string, topics ...string) (*Cluster, error) {
	cfg := sarama.NewConfig()
	cfg.Net.DialTimeout = PingTimeout
	cfg.Net.ReadTimeout = PingTimeout
	cfg.Net.WriteTimeout = PingTimeout
	cfg.Metadata.Retry.Max = 0
	cfg.Metadata.Timeout = PingTimeout

	client, err := sarama.NewClient(brokers, cfg)
	if err != nil {
		return nil, ErrKafkaUnavailable.WithCause(err)
	}

	return &Cluster{client: client, topics: topics}, nil
}

func (c *Cluster) Ping(ctx context.Context) error {
	// A port that accepts but never answers stalls sarama for one read timeout per request of the handshake.
	ctx, cancel := context.WithTimeout(ctx, PingTimeout)
	defer cancel()

	refreshed := make(chan error, 1)

	go func() { refreshed <- c.client.RefreshMetadata(c.topics...) }()

	select {
	case err := <-refreshed:
		if err != nil {
			return ErrKafkaUnavailable.WithCause(err)
		}

		return nil
	case <-ctx.Done():
		return ErrKafkaUnavailable.WithCause(ctx.Err())
	}
}

func (c *Cluster) Close() error {
	return c.client.Close()
}
