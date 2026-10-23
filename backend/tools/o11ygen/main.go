package main

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/dashboards"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/logrules"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/rules"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/slo"
)

func main() {
	root := flag.String("root", ".", "backend directory the artifact paths are relative to")

	flag.Parse()

	if err := generate(*root, services); err != nil {
		fmt.Fprintln(os.Stderr, "o11ygen:", errorReport(err))
		os.Exit(1)
	}
}

func generate(root string, declared []service.Service) error {
	if err := service.Validate(declared); err != nil {
		return err
	}

	return artifact.Generate(root, declared, append(loadedKinds(), rules.FixtureKind())...)
}

func loadedKinds() []artifact.Kind {
	return []artifact.Kind{dashboards.Kind(), rules.Kind(), logrules.Kind(), slo.SpecKind(), slo.RuleKind()}
}

func errorReport(err error) string {
	parts := append([]string{err.Error()}, errors.Reasons(err)...)

	attributes := errors.Attributes(err)
	for _, key := range slices.Sorted(maps.Keys(attributes)) {
		parts = append(parts, key+"="+attributes[key])
	}

	return strings.Join(parts, ": ")
}
