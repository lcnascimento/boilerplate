package notes

import "time"

type Option func(*Repository)

// WithClock sets the ce_time of NoteDeleted.
func WithClock(now func() time.Time) Option {
	return func(r *Repository) {
		r.now = now
	}
}
