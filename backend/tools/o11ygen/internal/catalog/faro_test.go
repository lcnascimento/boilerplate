package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/catalog"
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

func TestFaroProbes_AskLokiForThePageLoadSignalsOfEveryBrowserProcess(t *testing.T) {
	probes := catalog.FaroProbes([]service.Service{
		{Name: "web", Runtime: service.Node, Processes: []service.Process{service.Daemon("web", service.API, service.Browser)}},
		{Name: "orders", Processes: []service.Process{service.Daemon("orders-api", service.API)}},
	}, "15m")

	stream := `{service_namespace="boilerplate", service_name=~"web"}`

	assert.Equal(t, []artifact.Query{
		{
			Language: artifact.LogQL,
			Expr:     `sum(count_over_time(` + stream + ` | kind="event" [15m]))`,
			Origin:   "faro event of web",
		},
		{
			Language: artifact.LogQL,
			Expr:     `sum(count_over_time(` + stream + ` | kind="measurement" [15m]))`,
			Origin:   "faro measurement of web",
		},
	}, probes)

	for _, probe := range probes {
		assert.NoError(t, catalog.Check(probe), probe.Origin)
	}
}

func TestFaroProbes_AskNothingWithoutABrowserProcess(t *testing.T) {
	assert.Empty(t, catalog.FaroProbes([]service.Service{
		{Name: "orders", Processes: []service.Process{service.Daemon("orders-api", service.API)}},
	}, "15m"))
}
