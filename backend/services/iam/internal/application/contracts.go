package application

import "context"

type ChangesRepository interface {
	Translate(ctx context.Context, limit int32) (int, error)
}
