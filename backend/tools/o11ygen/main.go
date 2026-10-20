package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/dashboards"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

func main() {
	root := flag.String("root", ".", "backend directory the artifact paths are relative to")

	flag.Parse()

	if err := generate(*root, services); err != nil {
		fmt.Fprintln(os.Stderr, "o11ygen:", strings.Join(append([]string{err.Error()}, errors.Reasons(err)...), ": "))
		os.Exit(1)
	}
}

func generate(root string, declared []service.Service) error {
	if err := service.Validate(declared); err != nil {
		return err
	}

	return artifact.Generate(root, declared, dashboards.Kind())
}
