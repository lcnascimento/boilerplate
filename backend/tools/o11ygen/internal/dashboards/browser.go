package dashboards

import (
	"strconv"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

const browserStatWidth = 4

// The good and poor boundaries are the ones web.dev publishes for each Core Web Vital.
var webVitals = []struct {
	vital       catalog.WebVital
	title       string
	description string
	unit        string
	good, poor  float64
}{
	{catalog.LCP, "LCP", "Largest Contentful Paint: quando o maior elemento da página aparece.", "ms", 2500, 4000},
	{catalog.INP, "INP", "Interaction to Next Paint: quanto a página demora para responder a uma interação.", "ms", 200, 500},
	{catalog.CLS, "CLS", "Cumulative Layout Shift: quanto o layout se desloca sem o usuário pedir.", "none", 0.1, 0.25},
	{catalog.FCP, "FCP", "First Contentful Paint: quando o primeiro conteúdo aparece.", "ms", 1800, 3000},
	{catalog.TTFB, "TTFB", "Time to First Byte: quanto o primeiro byte do documento demora a chegar.", "ms", 800, 1800},
}

func browserSection(_ *service.Service, processes []service.Process) section {
	names := processNames(processes)
	p75 := func(vital catalog.WebVital) query {
		measurement := catalog.FaroStream(catalog.FaroMeasurement, names) +
			` | ` + catalog.FaroTypeLabel + `=` + strconv.Quote(catalog.FaroWebVitalsType) +
			` | ` + vital.ValueLabel() + `!=""`

		return query{
			expr:   "quantile_over_time(0.75, " + measurement + " | unwrap " + vital.ValueLabel() + " [$__auto]) by ()",
			legend: string(vital),
		}
	}
	pageLoads := "sum(count_over_time(" + catalog.FaroStream(catalog.FaroEvent, names) +
		` | ` + catalog.FaroEventNameLabel + `=` + strconv.Quote(catalog.FaroPageLoadEvent) + " [$__auto]))"
	jsErrors := func(by string) string {
		return "sum" + by + "(count_over_time(" + catalog.FaroStream(catalog.FaroException, names) + " [$__auto]))"
	}

	panels := make([]panel, 0, len(webVitals)+5)

	for _, v := range webVitals {
		panels = append(panels, &single{
			title:       v.title,
			description: v.description + " Percentil 75 no intervalo do dashboard.",
			unit:        v.unit,
			width:       browserStatWidth,
			logQL:       true,
			queries:     []query{p75(v.vital)},
			steps:       []step{{color: "green"}, at(v.good, "yellow"), at(v.poor, "red")},
		})
	}

	return section{title: string(service.Browser), panels: append(panels,
		&single{
			title:       "Erros de JS por page load",
			description: "Exceções que o Faro capturou no browser, por page load, no intervalo do dashboard.",
			unit:        "short",
			width:       browserStatWidth,
			logQL:       true,
			queries:     []query{{expr: "(" + jsErrors("") + " or vector(0)) / " + pageLoads}},
		},
		&graph{
			title:       "Web Vitals de tempo (p75)",
			description: "Percentil 75 de LCP, INP, FCP e TTFB ao longo do tempo.",
			unit:        "ms",
			width:       halfWidth,
			logQL:       true,
			queries:     []query{p75(catalog.LCP), p75(catalog.INP), p75(catalog.FCP), p75(catalog.TTFB)},
		},
		&graph{
			title:       "CLS (p75)",
			description: "Percentil 75 do Cumulative Layout Shift ao longo do tempo.",
			unit:        "none",
			width:       halfWidth,
			logQL:       true,
			queries:     []query{p75(catalog.CLS)},
		},
		&graph{
			title:       "Page loads",
			description: "Carregamentos completos de página que o browser reportou; navegações do lado do cliente não contam.",
			unit:        "short",
			width:       halfWidth,
			logQL:       true,
			queries:     []query{{expr: pageLoads, legend: "page loads"}},
		},
		&graph{
			title:       "Erros de JS por tipo",
			description: "Exceções capturadas no browser, por tipo de erro.",
			unit:        "short",
			width:       halfWidth,
			logQL:       true,
			queries:     []query{{expr: jsErrors(" by (" + catalog.FaroTypeLabel + ") "), legend: "{{type}}"}},
		},
	)}
}
