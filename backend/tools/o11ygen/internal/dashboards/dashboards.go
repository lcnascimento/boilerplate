package dashboards

import (
	"encoding/json"

	"github.com/grafana/grafana-foundation-sdk/go/dashboard"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

func Kind() artifact.Kind {
	return artifact.Kind{Dir: "config/grafana/provisioning/dashboards", Glob: "*.json", Build: build, Queries: queries}
}

func build(_ []service.Service) ([]artifact.File, error) {
	boards := []struct {
		name    string
		builder *dashboard.DashboardBuilder
	}{
		{"http", httpDashboard()},
		{"logs", logsDashboard()},
		{"messaging", messagingDashboard()},
		// example:begin
		{"notes-outbox", notesOutboxDashboard()},
		// example:end
		{"overview", overviewDashboard()},
		{"traces", tracesDashboard()},
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
