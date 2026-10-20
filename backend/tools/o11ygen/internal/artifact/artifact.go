package artifact

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

type File struct {
	Name    string
	Content []byte
}

type Language string

const (
	PromQL Language = "PromQL"
	LogQL  Language = "LogQL"
)

type Query struct {
	Language Language
	Expr     string
	Origin   string
}

type Kind struct {
	Dir     string
	Glob    string
	Build   func(services []service.Service) ([]File, error)
	Queries func(content []byte) ([]Query, error)
}

// World-readable: the observability containers read these files through bind mounts as another user.
const (
	dirMode  = 0o755
	fileMode = 0o644
)

// Split so that clean-example and ci-template, which grep the repository for fence markers, skip this line.
var fenceMarkers = [][]byte{[]byte("example" + ":begin"), []byte("example" + ":end")}

func Generate(root string, services []service.Service, kinds ...Kind) error {
	for _, kind := range kinds {
		files, err := kind.Build(services)
		if err != nil {
			return errors.New("building %s/%s", kind.Dir, kind.Glob).WithCause(err)
		}

		if err := write(filepath.Join(root, kind.Dir), kind.Glob, files); err != nil {
			return err
		}
	}

	return nil
}

func write(dir, glob string, files []File) error {
	keep := map[string]bool{}

	for _, file := range files {
		if err := check(glob, file); err != nil {
			return errors.New("generating %s", filepath.Join(dir, file.Name)).WithCause(err)
		}

		keep[file.Name] = true
	}

	if err := os.MkdirAll(dir, dirMode); err != nil {
		return errors.New("creating %s", dir).WithCause(err)
	}

	stale, err := filepath.Glob(filepath.Join(dir, glob))
	if err != nil {
		return errors.New("listing %s", filepath.Join(dir, glob)).WithCause(err)
	}

	for _, path := range stale {
		if keep[filepath.Base(path)] {
			continue
		}

		if err := os.Remove(path); err != nil {
			return errors.New("removing %s", path).WithCause(err)
		}
	}

	for _, file := range files {
		path := filepath.Join(dir, file.Name)
		if err := os.WriteFile(path, file.Content, fileMode); err != nil {
			return errors.New("writing %s", path).WithCause(err)
		}
	}

	return nil
}

func check(glob string, file File) error {
	if matched, err := filepath.Match(glob, file.Name); err != nil || !matched || filepath.Base(file.Name) != file.Name {
		return errors.New("the file name does not match %s, so a later run would not clean it up", glob)
	}

	for _, marker := range fenceMarkers {
		if bytes.Contains(file.Content, marker) {
			return errors.New("the content carries %q: example fences belong in the generator code only", marker)
		}
	}

	return nil
}
