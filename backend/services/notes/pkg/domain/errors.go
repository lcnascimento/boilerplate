package domain

import "github.com/lcnascimento/go-kit/errors"

var (
	// ErrInvalidInput carries one cause per validation failure, exposed as details.reasons.
	ErrInvalidInput = errors.New("invalid input").
			WithCode("INVALID_INPUT").
			WithKind(errors.KindInvalidInput)

	ErrNoteNotFound = errors.New("note not found").
			WithCode("NOTE_NOT_FOUND").
			WithKind(errors.KindNotFound)

	ErrInvalidNoteID = errors.New("invalid note id").
				WithCode("INVALID_NOTE_ID").
				WithKind(errors.KindInvalidInput)
)
