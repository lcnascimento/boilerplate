package frontend

type Option func(*Server)

func WithShutdown(shutdown func()) Option {
	return func(s *Server) {
		s.shutdown = shutdown
	}
}
