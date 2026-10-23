package translator

import "github.com/lcnascimento/go-kit/errors"

var (
	ErrStart = errors.New("failed to start the identity change translator")
	ErrStop  = errors.New("failed to stop the identity change translator")
)
