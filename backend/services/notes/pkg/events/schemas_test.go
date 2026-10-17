package events_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/lcnascimento/boilerplate/backend/services/notes/pkg/events"
)

func TestSubjects_OnePerProtoFile(t *testing.T) {
	t.Parallel()

	var protoFiles int

	err := fs.WalkDir(os.DirFS("proto"), ".", func(path string, _ fs.DirEntry, err error) error {
		if filepath.Ext(path) == ".proto" {
			protoFiles++
		}

		return err
	})
	require.NoError(t, err)

	got, err := events.Subjects()

	require.NoError(t, err)
	require.Len(t, got, protoFiles)
	require.ElementsMatch(t, []string{fullName(&events.NoteCreated{}), fullName(&events.NoteDeleted{})}, got)
}

func fullName(m proto.Message) string {
	return string(m.ProtoReflect().Descriptor().FullName())
}
