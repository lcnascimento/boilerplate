package outbox_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/mock/gomock"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/application/internal/mocks"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/application/outbox"
)

var metrics = sdkmetric.NewManualReader()

func TestMain(m *testing.M) {
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(metrics)))

	os.Exit(m.Run())
}

func newApplication(t *testing.T) (*outbox.Application, *mocks.MockOutboxRepository) {
	t.Helper()

	repo := mocks.NewMockOutboxRepository(gomock.NewController(t))

	return outbox.New(repo), repo
}

func TestApplication_PurgeKeepsSevenDays(t *testing.T) {
	t.Parallel()

	app, repo := newApplication(t)
	repo.EXPECT().Purge(gomock.Any(), 168*time.Hour).Return(int64(3), nil)

	require.NoError(t, app.Purge(t.Context()))
}

func TestApplication_PurgePropagatesTheRepositoryError(t *testing.T) {
	t.Parallel()

	app, repo := newApplication(t)
	repo.EXPECT().Purge(gomock.Any(), gomock.Any()).Return(int64(0), errors.ErrMock)

	err := app.Purge(t.Context())

	require.ErrorIs(t, err, outbox.ErrPurge)
	require.ErrorIs(t, err, errors.ErrMock)
}

func TestApplication_OldestPendingAgeComesFromTheRepository(t *testing.T) {
	t.Parallel()

	app, repo := newApplication(t)
	repo.EXPECT().OldestPendingAge(gomock.Any()).Return(90*time.Second, nil)

	age, err := app.OldestPendingAge(t.Context())

	require.NoError(t, err)
	require.Equal(t, 90*time.Second, age)
}

func TestApplication_OldestPendingAgePropagatesTheRepositoryError(t *testing.T) {
	t.Parallel()

	app, repo := newApplication(t)
	repo.EXPECT().OldestPendingAge(gomock.Any()).Return(time.Duration(0), errors.ErrMock)

	_, err := app.OldestPendingAge(t.Context())

	require.ErrorIs(t, err, outbox.ErrOldestPendingAge)
	require.ErrorIs(t, err, errors.ErrMock)
}

func TestApplication_PurgeCountsRunsByOutcomeAndPurgedMessages(t *testing.T) {
	app, repo := newApplication(t)
	gomock.InOrder(
		repo.EXPECT().Purge(gomock.Any(), gomock.Any()).Return(int64(4), nil),
		repo.EXPECT().Purge(gomock.Any(), gomock.Any()).Return(int64(1), errors.ErrMock),
		repo.EXPECT().Purge(gomock.Any(), gomock.Any()).Return(int64(0), errors.ErrMock),
	)

	before := collect(t)

	require.NoError(t, app.Purge(t.Context()))
	require.Error(t, app.Purge(t.Context()))
	require.Error(t, app.Purge(t.Context()))

	after := collect(t)

	require.Equal(t, int64(1), after["notes.outbox.purge.runs{outcome=success}"]-before["notes.outbox.purge.runs{outcome=success}"])
	require.Equal(t, int64(2), after["notes.outbox.purge.runs{outcome=failure}"]-before["notes.outbox.purge.runs{outcome=failure}"])
	require.Equal(t, int64(5), after["notes.outbox.purged.messages"]-before["notes.outbox.purged.messages"])
}

// collect keys each counter data point by name and outcome, as in notes.outbox.purge.runs{outcome=failure}.
func collect(t *testing.T) map[string]int64 {
	t.Helper()

	var collected metricdata.ResourceMetrics
	require.NoError(t, metrics.Collect(t.Context(), &collected))

	out := map[string]int64{}

	for _, scope := range collected.ScopeMetrics {
		for _, m := range scope.Metrics {
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}

			for _, point := range sum.DataPoints {
				out[m.Name+labels(point.Attributes)] = point.Value
			}
		}
	}

	return out
}

func labels(set attribute.Set) string {
	if outcome, ok := set.Value("outcome"); ok {
		return "{outcome=" + outcome.AsString() + "}"
	}

	return ""
}
