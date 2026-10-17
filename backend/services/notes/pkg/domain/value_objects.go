package domain

import (
	"github.com/google/uuid"

	"github.com/lcnascimento/go-kit/util"
)

const ContextKeyNoteID string = "note_id"

const noteIDPrefix string = "note"

type NoteID struct {
	util.CanonicalID
}

func NewNoteID() NoteID {
	return NoteID{util.NewCanonicalID(noteIDPrefix)}
}

func NoteIDFromUUID(u uuid.UUID) NoteID {
	return NoteID{util.CanonicalIDFromUUID(noteIDPrefix, u)}
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
