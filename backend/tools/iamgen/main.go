package main

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
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
	out := flag.String("out", "", "path of the generated authz.gen.ts")

	flag.Parse()

	if err := generate(*out); err != nil {
		fmt.Fprintln(os.Stderr, "iamgen:", errorReport(err))
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

func errorReport(err error) string {
	parts := append([]string{err.Error()}, errors.Reasons(err)...)

	attributes := errors.Attributes(err)
	for _, key := range slices.Sorted(maps.Keys(attributes)) {
		parts = append(parts, key+"="+attributes[key])
	}

	return strings.Join(parts, ": ")
}
