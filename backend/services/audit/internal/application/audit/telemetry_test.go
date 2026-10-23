package audit_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/mock/gomock"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/application/audit"
	"github.com/lcnascimento/boilerplate/backend/services/audit/internal/application/internal/mocks"
)

var spans = tracetest.NewSpanRecorder()

func TestMain(m *testing.M) {
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans)))

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
