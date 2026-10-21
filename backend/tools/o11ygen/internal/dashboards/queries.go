package dashboards

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/artifact"
)

type jsonDatasource struct {
	Type string `json:"type"`
}

type jsonTarget struct {
	RefID      string          `json:"refId"` //nolint:tagliatelle // Grafana's dashboard schema.
	Datasource *jsonDatasource `json:"datasource"`
	Expr       string          `json:"expr"`
}

type jsonPanel struct {
	ID         int             `json:"id"`
	Title      string          `json:"title"`
	Datasource *jsonDatasource `json:"datasource"`
	Targets    []jsonTarget    `json:"targets"`
	Panels     []jsonPanel     `json:"panels"`
}

type jsonVariable struct {
	Type       string          `json:"type"`
	Name       string          `json:"name"`
	Datasource *jsonDatasource `json:"datasource"`
	Query      json.RawMessage `json:"query"`
}

type jsonDashboard struct {
	Templating struct {
		List []jsonVariable `json:"list"`
	} `json:"templating"`
	Panels []jsonPanel `json:"panels"`
}

var (
	languages = map[string]artifact.Language{
		prometheusDS.kind: artifact.PromQL,
		lokiDS.kind:       artifact.LogQL,
	}
	withoutCatalogLanguage = map[string]bool{tempoDS.kind: true, "grafana": true}

	labelValues = regexp.MustCompile(`^label_values\((?:(.+),\s*)?(\w+)\)$`)
)

func queries(content []byte) ([]artifact.Query, error) {
	var board jsonDashboard
	if err := json.Unmarshal(content, &board); err != nil {
		return nil, errors.New("decoding dashboard").WithCause(err)
	}

	var out []artifact.Query

	for _, variable := range board.Templating.List {
		query, ok, err := variableQuery(variable)
		if err != nil {
			return nil, err
		}

		if ok {
			out = append(out, query)
		}
	}

	return panelQueries(out, board.Panels)
}

func languageOf(ds *jsonDatasource, origin string) (artifact.Language, bool, error) {
	if ds == nil {
		return "", false, errors.New("%s has a query but no datasource", origin)
	}

	if withoutCatalogLanguage[ds.Type] {
		return "", false, nil
	}

	language, ok := languages[ds.Type]
	if !ok {
		return "", false, errors.New("%s: datasource type %q has no query language registered in the dashboards kind", origin, ds.Type)
	}

	return language, true, nil
}

func variableQuery(variable jsonVariable) (artifact.Query, bool, error) {
	if variable.Type != "query" {
		return artifact.Query{}, false, nil
	}

	origin := "variable " + variable.Name

	language, ok, err := languageOf(variable.Datasource, origin)
	if err != nil || !ok {
		return artifact.Query{}, false, err
	}

	var raw any
	if err := json.Unmarshal(variable.Query, &raw); err != nil {
		return artifact.Query{}, false, errors.New("decoding %s", origin).WithCause(err)
	}

	text, _ := raw.(string)
	if object, isObject := raw.(map[string]any); isObject {
		text, _ = object["query"].(string)
	}

	match := labelValues.FindStringSubmatch(text)
	if match == nil {
		return artifact.Query{}, false, errors.New("%s: only label_values(...) queries are checked, got %q", origin, text)
	}

	selector, label := match[1], match[2]

	expr := fmt.Sprintf("group by (%s) (%s)", label, selector)
	if selector == "" {
		selector = fmt.Sprintf("{%s=~\".+\"}", label)
		expr = fmt.Sprintf("group by (%s) (%s)", label, selector)
	}

	if language == artifact.LogQL {
		expr = fmt.Sprintf("sum by (%s) (count_over_time(%s[1m]))", label, selector)
	}

	return artifact.Query{Language: language, Expr: expr, Origin: origin}, true, nil
}

func panelQueries(out []artifact.Query, panels []jsonPanel) ([]artifact.Query, error) {
	for _, panel := range panels {
		for _, target := range panel.Targets {
			if target.Expr == "" {
				continue
			}

			ds := target.Datasource
			if ds == nil {
				ds = panel.Datasource
			}

			origin := fmt.Sprintf("panel %d %q target %s", panel.ID, panel.Title, target.RefID)

			language, ok, err := languageOf(ds, origin)
			if err != nil {
				return nil, err
			}

			if ok {
				out = append(out, artifact.Query{Language: language, Expr: target.Expr, Origin: origin})
			}
		}

		var err error
		if out, err = panelQueries(out, panel.Panels); err != nil {
			return nil, err
		}
	}

	return out, nil
}
