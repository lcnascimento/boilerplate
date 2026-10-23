package metrictest_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/metrictest"
)

func TestCounterBy_SumsTheCounterByTheAttribute(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	counter, err := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)).Meter("test").Int64Counter("things.created")
	require.NoError(t, err)

	counter.Add(t.Context(), 2, metric.WithAttributes(attribute.String("kind", "a"), attribute.String("by", "x")))
	counter.Add(t.Context(), 3, metric.WithAttributes(attribute.String("kind", "a"), attribute.String("by", "y")))
	counter.Add(t.Context(), 4, metric.WithAttributes(attribute.String("kind", "b")))

	require.Equal(t, map[string]int64{"a": 5, "b": 4}, metrictest.CounterBy(t, reader, "things.created", "kind"))
	require.Equal(t, int64(9), metrictest.Counter(t, reader, "things.created"))
	require.Zero(t, metrictest.Counter(t, reader, "things.deleted"))
}
