package transport

import "context"

type ChangesService interface {
	Translate(ctx context.Context) (int, error)
}

type Database interface {
	Ping(ctx context.Context) error
}
