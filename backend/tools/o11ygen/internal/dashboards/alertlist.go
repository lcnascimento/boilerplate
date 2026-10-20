package dashboards

import (
	"strings"

	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
)

const everyAlert = ""

func alertsNamed(names []string) string {
	return `{alertname=~"` + strings.Join(names, "|") + `"}`
}

func alertList(id uint32, title, description, filter string, withNormal bool) *dashboard.PanelBuilder {
	return dashboard.NewPanelBuilder().
		Type("alertlist").
		Id(id).
		Title(title).
		Description(description).
		Options(map[string]any{
			"viewMode":                 "list",
			"groupMode":                "default",
			"groupBy":                  []string{},
			"maxItems":                 50,
			"sortOrder":                1,
			"dashboardAlerts":          false,
			"alertName":                "",
			"alertInstanceLabelFilter": filter,
			"stateFilter": map[string]bool{
				"firing":     true,
				"pending":    true,
				"noData":     true,
				"normal":     withNormal,
				"error":      true,
				"recovering": true,
			},
			"showInstances": false,
		})
}
