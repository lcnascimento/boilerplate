package consumer

type Option func(*Consumer)

func WithShutdown(shutdown func()) Option {
	return func(c *Consumer) {
		c.shutdown = shutdown
	}
}
