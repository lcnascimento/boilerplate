package dashboards

import (
	"strconv"
	"strings"

	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/config"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const processVariable = "process"

type section struct {
	title     string
	collapsed bool
	panels    []panel
}

var runtimeDashboards = map[service.Runtime]struct {
	api      func(svc *service.Service, processes []service.Process) section
	runtime  func() section
	profiles bool
}{
	service.Go:   {api: goAPISection, runtime: goRuntimeSection, profiles: true},
	service.Node: {api: spanMetricsAPISection, runtime: nodeRuntimeSection},
}

func apiSection(svc *service.Service, processes []service.Process) section {
	return runtimeDashboards[svc.Runtime].api(svc, processes)
}

var componentSections = []struct {
	component service.Component
	section   func(svc *service.Service, processes []service.Process) section
}{
	{service.API, apiSection},
	{service.Consumer, consumerSection},
	{service.Worker, workerSection},
	{service.Browser, browserSection},
}

func serviceDashboard(svc *service.Service, declaresSLOs bool) *dashboard.DashboardBuilder {
	sections := []section{topSection(svc, declaresSLOs)}

	for _, c := range componentSections {
		if running := processesRunning(svc, c.component); len(running) > 0 {
			sections = append(sections, c.section(svc, running))
		}
	}

	sections = append(sections, runtimeDashboards[svc.Runtime].runtime())

	if hasDBPool(svc) {
		sections = append(sections, databaseSection())
	}

	sections = append(sections, logsSection())

	out := newDashboard(svc.Name).
		Description("SLOs, alertas, uma seção por componente, runtime, banco e logs do serviço "+svc.Name+".").
		Tags([]string{config.Namespace, "service"}).
		Readonly().
		Tooltip(dashboard.DashboardCursorSyncCrosshair).
		Timepicker(dashboard.NewTimePickerBuilder()).
		Refresh("30s").
		Time("now-1h", "now").
		Variables([]cog.Builder[dashboard.VariableModel]{
			datasourceVariable(tempoDS), datasourceVariable(prometheusDS), datasourceVariable(lokiDS),
			processVariableOf(svc),
		}).
		Annotations([]cog.Builder[dashboard.AnnotationQuery]{dashboard.NewAnnotationQueryBuilder().
			Name("Annotations & Alerts").
			Datasource(common.DataSourceRef{Type: cog.ToPtr("grafana"), Uid: cog.ToPtr("-- Grafana --")}).
			Hide(true).
			IconColor("rgba(0, 211, 255, 1)").
			Type("dashboard").
			BuiltIn(1)}).
		Links(drilldownLinks(svc))

	var id uint32

	next := func() uint32 {
		id++

		return id
	}

	for _, s := range sections {
		row := dashboard.NewRowBuilder(s.title).Id(next())

		if s.collapsed {
			for _, p := range s.panels {
				row.WithPanel(p.build(next()))
			}

			out.WithRow(row)

			continue
		}

		out.WithRow(row)

		for _, p := range s.panels {
			out.WithPanel(p.build(next()))
		}
	}

	return out
}

func processesRunning(svc *service.Service, component service.Component) []service.Process {
	var out []service.Process

	for _, p := range svc.Processes {
		if p.Runs(component) {
			out = append(out, p)
		}
	}

	return out
}

func hasDBPool(svc *service.Service) bool {
	for _, p := range svc.Processes {
		if p.DBPool() {
			return true
		}
	}

	return false
}

func processVariableOf(svc *service.Service) *dashboard.CustomVariableBuilder {
	names := processNames(longRunning(svc))

	return dashboard.NewCustomVariableBuilder(processVariable).
		Label("Processo").
		Values(dashboard.StringOrMap{String: cog.ToPtr(strings.Join(names, ","))}).
		Current(dashboard.VariableOption{
			Selected: cog.ToPtr(true),
			Text:     dashboard.StringOrArrayOfString{ArrayOfString: []string{"All"}},
			Value:    dashboard.StringOrArrayOfString{ArrayOfString: []string{"$__all"}},
		}).
		Multi(true).
		IncludeAll(true)
}

func processNames(processes []service.Process) []string {
	names := make([]string, 0, len(processes))
	for _, p := range processes {
		names = append(names, p.Name())
	}

	return names
}

func longRunning(svc *service.Service) []service.Process {
	var out []service.Process

	for _, p := range svc.Processes {
		if p.LongRunning() {
			out = append(out, p)
		}
	}

	return out
}

func job(processes []service.Process) string {
	return `job=~` + strconv.Quote(config.Namespace+"/("+strings.Join(processNames(processes), "|")+")")
}

const selectedJob = `job=~"` + config.Namespace + `/$` + processVariable + `"`
