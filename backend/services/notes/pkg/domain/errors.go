package domain

import "github.com/lcnascimento/go-kit/errors"

var (
	ErrInvalidInput = errors.New("invalid input").
			WithCode("INVALID_INPUT").
			WithKind(errors.KindInvalidInput)

	ErrNoteNotFound = errors.New("note not found").
			WithCode("NOTE_NOT_FOUND").
			WithKind(errors.KindNotFound)

	ErrOrganizationDeleted = errors.New("the organization was deleted").
				WithCode("ORGANIZATION_DELETED").
				WithKind(errors.KindUnauthorized)

	ErrInvalidNoteID = errors.New("invalid note id").
				WithCode("INVALID_NOTE_ID").
				WithKind(errors.KindInvalidInput)
)
