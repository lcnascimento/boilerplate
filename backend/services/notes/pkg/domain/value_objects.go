package domain

import (
	"unicode/utf8"

	"github.com/lcnascimento/go-kit/errors"
	"github.com/lcnascimento/go-kit/util"
)

const ContextKeyNoteID string = "note_id"

const noteIDPrefix string = "note"

// NoteTitleMaxLength is the title limit in characters (runes).
// models.CreateNoteRequest repeats it in a validate tag; a test there keeps both in sync.
const NoteTitleMaxLength = 120

type NoteID struct {
	util.CanonicalID
}

func NewNoteID() NoteID {
	return NoteID{util.NewCanonicalID(noteIDPrefix)}
}

func ParseNoteID(s string) (NoteID, error) {
	id, err := util.ParseCanonicalID(noteIDPrefix, s)
	if err != nil {
		return NoteID{}, ErrInvalidNoteID.WithCause(err)
	}

	return NoteID{id}, nil
}

func MustParseNoteID(s string) NoteID {
	id, err := ParseNoteID(s)
	if err != nil {
		panic(err)
	}

	return id
}

func (id NoteID) Validate() error {
	if id.IsZero() {
		return ErrInvalidNoteID.WithCause(errors.New("note id is empty"))
	}

	return nil
}

// MarshalText has a value receiver so a NoteID inside a non-addressable value
// (a Note passed by value, a map entry) still encodes as its prefixed string.
func (id NoteID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

// UnmarshalText overrides the promoted CanonicalID implementation, which
// cannot parse into a zero value (its prefix is empty until set).
func (id *NoteID) UnmarshalText(b []byte) error {
	parsed, err := ParseNoteID(string(b))
	if err != nil {
		return err
	}

	*id = parsed

	return nil
}

type NoteTitle string

func (t NoteTitle) Validate() error {
	if t == "" {
		return ErrInvalidNoteTitle.WithCause(errors.New("title is required"))
	}

	if utf8.RuneCountInString(string(t)) > NoteTitleMaxLength {
		return ErrInvalidNoteTitle.WithCause(errors.New("title must be at most 120 characters"))
	}

	return nil
}

func (t NoteTitle) String() string {
	return string(t)
}
