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
	prometheusDS: "Prometheus data source",
	lokiDS:       "Loki data source",
	tempoDS:      "Tempo data source",
}

func (ds datasource) variable() string {
	return string(ds) + "_datasource"
}

func datasourceRef(ds datasource) common.DataSourceRef {
	return common.DataSourceRef{Type: cog.ToPtr(string(ds)), Uid: cog.ToPtr("${" + ds.variable() + "}")}
}

func datasourceVariable(ds datasource) *dashboard.DatasourceVariableBuilder {
	return dashboard.NewDatasourceVariableBuilder(ds.variable()).
		Label(datasourceLabels[ds]).
		Type(string(ds))
}
