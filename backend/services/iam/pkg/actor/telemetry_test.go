package actor_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/lcnascimento/go-kit/o11y/log"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/actor/actortest"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/roles"
)

const authenticationFailures = "iam.actor.authentication.failures"

var (
	spans   = tracetest.NewSpanRecorder()
	metrics = sdkmetric.NewManualReader()
	tracer  = otel.Tracer("actor_test")
)

func TestMain(m *testing.M) {
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans)))
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(metrics)))

	os.Exit(m.Run())
}

// inSpan runs fn under a request span, like the go-kit Telemetry middleware opens, and returns that span's attributes.
func inSpan(t *testing.T, fn func(ctx context.Context)) attribute.Set {
	t.Helper()

	ctx, span := tracer.Start(t.Context(), t.Name(), trace.WithSpanKind(trace.SpanKindServer))
	fn(ctx)
	span.End()

	for _, s := range spans.Ended() {
		if s.SpanContext().SpanID() == span.SpanContext().SpanID() {
			return attribute.NewSet(s.Attributes()...)
		}
	}

	t.Fatal("the request span did not end")

	return attribute.Set{}
}

func TestMiddleware_PutsTheActorIDsOnTheRequestSpanAndLogs(t *testing.T) {
	issuer := actortest.NewIssuer(t)
	a := &actor.Actor{
		UserID:         mustUserID(t, "user_5d2a7c10-3e4f-4b6a-8c9d-0e1f2a3b4c5d"),
		OrganizationID: acmeID(t),
		Role:           roles.Owner,
	}

	var logs bytes.Buffer

	requestLogger := log.MustNewLogger("handler", log.WithLogHandler(slog.NewJSONHandler(&logs, nil)))
	mw := newMiddleware(t, issuer.Config())

	span := inSpan(t, func(ctx context.Context) {
		serve(ctx, mw, bearer(issuer.Mint(a)), func(r *http.Request) {
			requestLogger.Info(r.Context(), "listing notes")
		})
	})

	require.Equal(t, "user_5d2a7c10-3e4f-4b6a-8c9d-0e1f2a3b4c5d", stringAttr(span, "user.id"))
	require.Equal(t, "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f", stringAttr(span, "organization.id"))

	var record struct {
		Bag map[string]string `json:"bag"`
	}

	require.NoError(t, json.Unmarshal(logs.Bytes(), &record))
	require.Equal(t, map[string]string{
		"user.id":         "user_5d2a7c10-3e4f-4b6a-8c9d-0e1f2a3b4c5d",
		"organization.id": "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f",
	}, record.Bag)
}

func failures(t *testing.T, reason string) int64 {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, metrics.Collect(t.Context(), &rm))

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != authenticationFailures {
				continue
			}

			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "%s is a %T", authenticationFailures, m.Data)

			for _, dp := range sum.DataPoints {
				if v, _ := dp.Attributes.Value("reason"); v.AsString() == reason {
					return dp.Value
				}
			}
		}
	}

	return 0
}

func stringAttr(set attribute.Set, key attribute.Key) string {
	v, _ := set.Value(key)

	return v.AsString()
}

func labelSets(t *testing.T, name string) []string {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, metrics.Collect(t.Context(), &rm))

	var out []string

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}

			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "%s is a %T", name, m.Data)

			for _, dp := range sum.DataPoints {
				out = append(out, dp.Attributes.Encoded(attribute.DefaultEncoder()))
			}
		}
	}

	return out
}

func TestNewMiddleware_SeedsEveryRejectionReasonSoTheSeriesExistBeforeTheFirstFailure(t *testing.T) {
	newMiddleware(t, actortest.NewIssuer(t).Config())

	reasons := labelSets(t, authenticationFailures)
	for _, reason := range []string{
		"missing_header", "malformed", "bad_signature", "unknown_kid", "bad_alg", "bad_issuer", "bad_audience", "expired",
		"missing_claim", "unknown_role",
	} {
		require.Contains(t, reasons, "reason="+reason)
	}
}

// example:begin

func TestNewMiddleware_SeedsEveryPermissionARoleLacksSoTheSeriesExistBeforeTheFirstDenial(t *testing.T) {
	newMiddleware(t, actortest.NewIssuer(t).Config())

	require.ElementsMatch(t, []string{
		"permission=note:delete,role=admin",
		"permission=note:create,role=viewer",
		"permission=note:delete,role=viewer",
	}, labelSets(t, "iam.actor.authorization.denials"))
}

// example:end
