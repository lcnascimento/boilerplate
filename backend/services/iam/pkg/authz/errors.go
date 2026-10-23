package authz

import "github.com/lcnascimento/go-kit/errors"

var ErrUnknownRole = errors.New("unknown role").
	WithCode("UNKNOWN_ROLE").
	WithKind(errors.KindInvalidInput)
