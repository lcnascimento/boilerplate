package dashboards

import (
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
)

type datasource string

const (
	prometheusDS datasource = "prometheus"
	lokiDS       datasource = "loki"
	tempoDS      datasource = "tempo"
)

var datasourceLabels = map[datasource]string{
	prometheusDS: "Prometheus",
	lokiDS:       "Loki",
	tempoDS:      "Tempo",
}

func datasourceRef(ds datasource) common.DataSourceRef {
	return common.DataSourceRef{Type: cog.ToPtr(string(ds)), Uid: cog.ToPtr("${" + string(ds) + "}")}
}

func datasourceVariable(ds datasource) *dashboard.DatasourceVariableBuilder {
	return dashboard.NewDatasourceVariableBuilder(string(ds)).
		Label(datasourceLabels[ds]).
		Type(string(ds))
}
