package eventbus

import (
	"io/fs"
	"path"

	"google.golang.org/protobuf/reflect/protoregistry"
)

type schema struct {
	subject string
	text    string
}

func MustProtos(embedded fs.FS) fs.FS {
	protos, err := fs.Sub(embedded, "proto")
	if err != nil {
		panic(ErrLoadSchemas.WithCause(err))
	}

	return protos
}

func loadSchemas(protos fs.FS) ([]schema, error) {
	var schemas []schema

	err := fs.WalkDir(protos, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || path.Ext(name) != ".proto" {
			return err
		}

		s, err := loadSchema(protos, name)
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
}

func loadSchema(protos fs.FS, name string) (schema, error) {
	text, err := fs.ReadFile(protos, name)
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

	return schema{subject: string(file.Messages().Get(0).FullName()), text: string(text)}, nil
}

func Subjects(protos fs.FS) ([]string, error) {
	schemas, err := loadSchemas(protos)
	if err != nil {
		return nil, err
	}

	subjects := make([]string, 0, len(schemas))
	for _, s := range schemas {
		subjects = append(subjects, s.subject)
	}

	return subjects, nil
}
