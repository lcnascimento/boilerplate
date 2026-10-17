package notes

import "time"

type Option func(*Application)

func WithClock(now func() time.Time) Option {
	return func(a *Application) {
		a.now = now
	}
}
