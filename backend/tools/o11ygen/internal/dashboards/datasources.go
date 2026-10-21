package dashboards

import (
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/common"
)

type datasource struct {
	kind string
	uid  string
}

var (
	prometheusDS = datasource{kind: "prometheus", uid: "prometheus"}
	lokiDS       = datasource{kind: "loki", uid: "loki"}
	tempoDS      = datasource{kind: "tempo", uid: "tempo"}
)

func datasourceRef(ds datasource) common.DataSourceRef {
	return common.DataSourceRef{Type: cog.ToPtr(ds.kind), Uid: cog.ToPtr(ds.uid)}
}
