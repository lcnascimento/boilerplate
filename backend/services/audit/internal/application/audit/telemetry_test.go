package audit_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/mock/gomock"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/application/audit"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/application/internal/mocks"
)

var (
	spans   = tracetest.NewSpanRecorder()
	metrics = sdkmetric.NewManualReader()
)

func TestMain(m *testing.M) {
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans)))
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(metrics)))

	os.Exit(m.Run())
}

func TestRecord_SpanCarriesTheCloudEventsAttributes(t *testing.T) {
	t.Parallel()

	entry := validEntry()
	entry.ID = "0199d9a8-6f1e-7c3a-9b1e-2f6d1c0a4b99"

	repo := mocks.NewMockAuditEntriesRepository(gomock.NewController(t))
	repo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)

	require.NoError(t, audit.New(repo, newValidator(t)).Record(t.Context(), entry))

	for _, span := range spans.Ended() {
		attrs := attribute.NewSet(span.Attributes()...)
		if id, _ := attrs.Value("cloudevents.event_id"); id.AsString() != entry.ID {
			continue
		}

		require.Equal(t, "RecordAuditEntry", span.Name())

		source, _ := attrs.Value("cloudevents.event_source")
		eventType, _ := attrs.Value("cloudevents.event_type")

		require.Equal(t, "iam", source.AsString())
		require.Equal(t, "boilerplate.iam.v1.UserDeleted", eventType.AsString())

		return
	}

	t.Fatal("no RecordAuditEntry span carries the event id")
}

func TestRecord_CountsTheRecordedEntriesByEventType(t *testing.T) {
	repo := mocks.NewMockAuditEntriesRepository(gomock.NewController(t))
	repo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil).Times(3)

	app := audit.New(repo, newValidator(t))
	before := recordedByType(t)

	for _, eventType := range []string{"boilerplate.iam.v1.MemberAdded", "boilerplate.iam.v1.MemberAdded", "boilerplate.iam.v1.UserCreated"} {
		entry := validEntry()
		entry.Type = eventType
		require.NoError(t, app.Record(t.Context(), entry))
	}

	after := recordedByType(t)
	require.Equal(t, int64(2), after["boilerplate.iam.v1.MemberAdded"]-before["boilerplate.iam.v1.MemberAdded"])
	require.Equal(t, int64(1), after["boilerplate.iam.v1.UserCreated"]-before["boilerplate.iam.v1.UserCreated"])
}

func recordedByType(t *testing.T) map[string]int64 {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, metrics.Collect(t.Context(), &rm))

	counts := map[string]int64{}

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "audit.entries.recorded" {
				continue
			}

			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "audit.entries.recorded is a %T", m.Data)

			for _, dp := range sum.DataPoints {
				eventType, _ := dp.Attributes.Value("cloudevents.event_type")
				counts[eventType.AsString()] = dp.Value
			}
		}
	}

	return counts
}
