package forwarder

type Option func(*Forwarder)

func WithShutdown(shutdown func()) Option {
	return func(f *Forwarder) {
		f.shutdown = shutdown
	}
}
