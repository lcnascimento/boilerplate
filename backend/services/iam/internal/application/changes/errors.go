package changes

import "github.com/lcnascimento/go-kit/errors"

var (
	ErrTranslateChanges = errors.New("failed to translate identity changes")
	ErrOldestPendingAge = errors.New("failed to read the age of the oldest identity change")
)
