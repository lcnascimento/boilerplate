package dashboards

import (
	"encoding/json"

	"github.com/grafana/grafana-foundation-sdk/go/dashboard"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/slo"
)

func Kind() artifact.Kind {
	return artifact.Kind{
		Dir: "config/grafana/provisioning/dashboards", Glob: "*.json", LoadedBy: artifact.Grafana, Build: build, Queries: queries,
	}
}

type board struct {
	name    string
	builder *dashboard.DashboardBuilder
}

func build(services []service.Service) ([]artifact.File, error) {
	boards := []board{
		{"kafka", kafkaDashboard()},
		{"overview", overviewDashboard()},
		{"postgres", postgresDashboard()},
	}

	taken := map[string]bool{}
	for _, b := range boards {
		taken[b.name] = true
	}

	for i := range services {
		svc := &services[i]
		if taken[svc.Name] {
			return nil, errors.New("service name %q is taken by a dashboard", svc.Name)
		}

		slos, err := slo.Declared(svc)
		if err != nil {
			return nil, err
		}

		boards = append(boards, board{svc.Name, serviceDashboard(svc, len(slos) > 0)})
	}

	files := make([]artifact.File, 0, len(boards))

	for _, board := range boards {
		built, err := board.builder.Build()
		if err != nil {
			return nil, errors.New("building dashboard %s", board.name).WithCause(err)
		}

		content, err := json.MarshalIndent(built, "", "  ")
		if err != nil {
			return nil, errors.New("encoding dashboard %s", board.name).WithCause(err)
		}

		files = append(files, artifact.File{Name: board.name + ".json", Content: append(content, '\n')})
	}

	return files, nil
}
