package main

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

var (
	quantileArg     = regexp.MustCompile(`^histogram_quantile\(([0-9.]+),`)
	percentileToken = regexp.MustCompile(` p\d+\b`)
)

func flatten(panels []boardPanel) []boardPanel {
	out := make([]boardPanel, 0, len(panels))

	for i := range panels {
		out = append(out, panels[i])
		out = append(out, flatten(panels[i].Panels)...)
	}

	return out
}

func TestGenerate_SplitsEveryLatencyGraphIntoAP50AP90AndAP99Panel(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, generate(root, []service.Service{ordersService, storefrontService}))

	files, err := filepath.Glob(filepath.Join(root, dashboardsDir, "*.json"))
	require.NoError(t, err)

	families := map[string][]string{}

	for _, file := range files {
		name := filepath.Base(file)
		board := readBoard(t, root, name)

		for _, panel := range flatten(board.Panels) {
			if panel.Type != "timeseries" {
				continue
			}

			var quantiles []string

			for _, target := range panel.Targets {
				if match := quantileArg.FindStringSubmatch(target.Expr); match != nil {
					quantiles = append(quantiles, match[1])
				}
			}

			if quantiles == nil {
				continue
			}

			token := strings.TrimSpace(percentileToken.FindString(panel.Title))
			family := name + ": " + percentileToken.ReplaceAllString(panel.Title, "")
			families[family] = append(families[family], token+"="+strings.Join(quantiles, ","))
		}
	}

	split := []string{"p50=0.5", "p90=0.9", "p99=0.99"}
	assert.Equal(t, map[string][]string{
		"kafka.json: Duração do processamento":  split,
		"orders.json: Latência":                 split,
		"orders.json: Latência por rota":        split,
		"orders.json: Duração do processamento": split,
		"orders.json: Duração do envio":         split,
		"storefront.json: Latência":             split,
		"storefront.json: Latência por rota":    split,
	}, families)
}
