package audit

import "github.com/lcnascimento/go-kit/errors"

var (
	ErrInvalidEnvelope = errors.New("Integration Event envelope is missing a required attribute").
				WithCode("INVALID_ENVELOPE").
				WithKind(errors.KindInvalidInput)

	ErrRecordEntry = errors.New("failed to record audit entry")
)
