package bundle_test

import (
	"testing"

	"go.uber.org/fx"

	"github.com/lcnascimento/boilerplate/backend/services/audit/bundle"
)

func TestDependencyGraphs(t *testing.T) {
	t.Parallel()

	graphs := map[string]func() []fx.Option{
		"consumer": bundle.ConsumerOptions,
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
