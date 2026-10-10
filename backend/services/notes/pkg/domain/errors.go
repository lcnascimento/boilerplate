package domain

import "github.com/lcnascimento/go-kit/errors"

// Details go in WithCause, never WithAttribute: in go-kit errors v1.9.0,
// WithAttribute writes into the sentinel's shared map, which races across requests.
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

	ErrInvalidNoteTitle = errors.New("invalid note title").
				WithCode("INVALID_NOTE_TITLE").
				WithKind(errors.KindInvalidInput)
)
