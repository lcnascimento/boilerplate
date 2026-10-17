package notes

import "time"

type Option func(*Repository)

func WithClock(now func() time.Time) Option {
	return func(r *Repository) {
		r.now = now
	}
}
