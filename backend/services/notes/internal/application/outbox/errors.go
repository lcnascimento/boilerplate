package outbox

import "github.com/lcnascimento/go-kit/errors"

var (
	ErrPurge            = errors.New("failed to purge the outbox")
	ErrOldestPendingAge = errors.New("failed to read the age of the oldest pending outbox message")
)
