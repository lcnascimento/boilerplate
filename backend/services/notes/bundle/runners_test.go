package bundle_test

import (
	"testing"

	"go.uber.org/fx"

	"github.com/lcnascimento/boilerplate/backend/services/notes/bundle"
)

func TestDependencyGraphs(t *testing.T) {
	t.Parallel()

	graphs := map[string]func() []fx.Option{
		"frontend server":  bundle.FrontendServerOptions,
		"worker forwarder": bundle.WorkerForwarderOptions,
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
