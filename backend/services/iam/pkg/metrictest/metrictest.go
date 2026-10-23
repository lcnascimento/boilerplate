package metrictest

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func CounterBy(t *testing.T, reader sdkmetric.Reader, name string, key attribute.Key) map[string]int64 {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &rm))

	counts := map[string]int64{}

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}

			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "%s is a %T", name, m.Data)

			for _, dp := range sum.DataPoints {
				value, _ := dp.Attributes.Value(key)
				counts[value.AsString()] += dp.Value
			}
		}
	}

	return counts
}

func Counter(t *testing.T, reader sdkmetric.Reader, name string) int64 {
	t.Helper()

	var total int64
	for _, count := range CounterBy(t, reader, name, "") {
		total += count
	}

	return total
}
