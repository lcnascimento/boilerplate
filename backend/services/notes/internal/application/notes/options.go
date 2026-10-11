package notes

import "time"

type Option func(*Application)

// WithClock replaces time.Now as the source of a note's CreatedAt.
func WithClock(now func() time.Time) Option {
	return func(a *Application) {
		a.now = now
	}
}
