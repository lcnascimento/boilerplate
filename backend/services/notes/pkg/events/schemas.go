package events

import (
	"embed"
	"io/fs"
	"path"
	"sync"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

//go:embed proto
var protoFS embed.FS

type schema struct {
	subject string
	text    string
	message protoreflect.MessageType
}

var loadSchemas = sync.OnceValues(func() ([]schema, error) {
	root, err := fs.Sub(protoFS, "proto")
	if err != nil {
		return nil, ErrLoadSchemas.WithCause(err)
	}

	var schemas []schema

	err = fs.WalkDir(root, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || path.Ext(name) != ".proto" {
			return err
		}

		s, err := loadSchema(root, name)
		if err != nil {
			return err
		}

		schemas = append(schemas, s)

		return nil
	})
	if err != nil {
		return nil, ErrLoadSchemas.WithCause(err)
	}

	return schemas, nil
})

func loadSchema(root fs.FS, name string) (schema, error) {
	text, err := fs.ReadFile(root, name)
	if err != nil {
		return schema{}, err
	}

	file, err := protoregistry.GlobalFiles.FindFileByPath(name)
	if err != nil {
		return schema{}, ErrLoadSchemas.WithAttribute("file", name).WithCause(err)
	}

	if file.Messages().Len() != 1 {
		return schema{}, ErrLoadSchemas.WithAttribute("file", name).WithCause(ErrNotOneMessagePerFile)
	}

	subject := file.Messages().Get(0).FullName()

	message, err := protoregistry.GlobalTypes.FindMessageByName(subject)
	if err != nil {
		return schema{}, ErrLoadSchemas.WithAttribute("file", name).WithCause(err)
	}

	return schema{subject: string(subject), text: string(text), message: message}, nil
}

func Subjects() ([]string, error) {
	schemas, err := loadSchemas()
	if err != nil {
		return nil, err
	}

	subjects := make([]string, 0, len(schemas))
	for _, s := range schemas {
		subjects = append(subjects, s.subject)
	}

	return subjects, nil
}
