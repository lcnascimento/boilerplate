package notes

import "github.com/lcnascimento/go-kit/errors"

var (
	ErrCreateNote = errors.New("failed to create note")
	ErrListNotes  = errors.New("failed to list notes")
	ErrGetNote    = errors.New("failed to get note")
	ErrDeleteNote = errors.New("failed to delete note")
)
