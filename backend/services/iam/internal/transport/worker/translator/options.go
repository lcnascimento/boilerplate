package translator

import "time"

type Option func(*Translator)

func WithPollInterval(interval time.Duration) Option {
	return func(t *Translator) {
		t.pollInterval = interval
	}
}
