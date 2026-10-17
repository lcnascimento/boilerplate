package forwarder

type Option func(*Forwarder)

// WithShutdown is called when the forwarder stops without Stop, so the process can exit.
func WithShutdown(shutdown func()) Option {
	return func(f *Forwarder) {
		f.shutdown = shutdown
	}
}
