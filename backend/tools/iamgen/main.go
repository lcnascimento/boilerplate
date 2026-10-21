package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lcnascimento/go-kit/errors"
)

const (
	dirMode  = 0o755
	fileMode = 0o644
)

var (
	ErrMissingOut = errors.New("missing -out")
	ErrWrite      = errors.New("could not write the generated file")
)

func main() {
	out := flag.String("out", "", "path of the generated roles.gen.ts")

	flag.Parse()

	if err := generate(*out); err != nil {
		fmt.Fprintln(os.Stderr, "iamgen:", strings.Join(append([]string{err.Error()}, errors.Reasons(err)...), ": "))
		os.Exit(1)
	}
}

func generate(out string) error {
	if out == "" {
		return ErrMissingOut
	}

	m, err := load()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(out), dirMode); err != nil {
		return ErrWrite.WithCause(err)
	}

	if err := os.WriteFile(out, render(m), fileMode); err != nil {
		return ErrWrite.WithCause(err)
	}

	return nil
}
