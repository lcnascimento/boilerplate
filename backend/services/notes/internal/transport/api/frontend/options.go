package frontend

type Option func(*Server)

// WithShutdown sets what the server calls when it stops serving on an error.
func WithShutdown(shutdown func()) Option {
	return func(s *Server) {
		s.shutdown = shutdown
	}
}
