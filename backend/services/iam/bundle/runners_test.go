package bundle_test

import (
	"testing"

	"go.uber.org/fx"

	"github.com/lcnascimento/boilerplate/backend/services/iam/bundle"
)

func TestDependencyGraphs(t *testing.T) {
	t.Parallel()

	graphs := map[string]func() []fx.Option{
		"worker": bundle.WorkerOptions,
	}

	for name, options := range graphs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := fx.ValidateApp(options()...); err != nil {
				t.Errorf("invalid dependency graph: %v", err)
			}
		})
	}
}
