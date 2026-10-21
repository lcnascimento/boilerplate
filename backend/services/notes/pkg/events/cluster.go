package events

import (
	"context"
	"time"

	"github.com/IBM/sarama"
)

// pingTimeout keeps a Ping inside the 2s budget of a health check.
const pingTimeout = time.Second

type Cluster struct {
	client sarama.Client
	topics []string
}

// DialCluster connects to the brokers; Ping then asks only for the metadata of the topics the process uses, since a
// leaderless partition elsewhere (__consumer_offsets while the broker starts) does not stop it.
func DialCluster(brokers []string, topics ...string) (*Cluster, error) {
	cfg := sarama.NewConfig()
	cfg.Net.DialTimeout = pingTimeout
	cfg.Net.ReadTimeout = pingTimeout
	cfg.Net.WriteTimeout = pingTimeout
	cfg.Metadata.Retry.Max = 0
	cfg.Metadata.Timeout = pingTimeout

	client, err := sarama.NewClient(brokers, cfg)
	if err != nil {
		return nil, ErrKafkaUnavailable.WithCause(err)
	}

	return &Cluster{client: client, topics: topics}, nil
}

// Ping asks a broker for the cluster metadata: the clients behind publishers and subscribers reconnect in the
// background without reporting a lost broker. A port that accepts but never answers stalls sarama for one read
// timeout per request of the handshake, so the deadline is enforced here.
func (c *Cluster) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
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
