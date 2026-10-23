package translator_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/mock/gomock"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/transport/internal/mocks"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/transport/worker/translator"
)

const (
	waitFor          = 5 * time.Second
	oldestPendingAge = "iam.identity_change.oldest_pending.age"
)

var metrics = sdkmetric.NewManualReader()

func TestMain(m *testing.M) {
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(metrics)))

	os.Exit(m.Run())
}

func TestTranslator_ReportsTheAgeOfTheOldestPendingChangeWhileRunning(t *testing.T) {
	changes := mocks.NewMockChangesService(gomock.NewController(t))
	changes.EXPECT().Translate(gomock.Any()).Return(0, nil).AnyTimes()
	changes.EXPECT().OldestPendingAge(gomock.Any()).Return(7*time.Minute+30*time.Second, nil)

	worker := translator.New(changes)
	require.NoError(t, worker.Start(t.Context()))

	require.Equal(t, []int64{450}, gaugeValues(t))

	require.NoError(t, worker.Stop(t.Context()))
	require.Empty(t, gaugeValues(t))
}

func TestTranslator_SkipsTheAgeWhenItCannotBeRead(t *testing.T) {
	changes := mocks.NewMockChangesService(gomock.NewController(t))
	changes.EXPECT().Translate(gomock.Any()).Return(0, nil).AnyTimes()
	changes.EXPECT().OldestPendingAge(gomock.Any()).Return(time.Duration(0), errors.ErrMock)

	start(t, translator.New(changes))

	require.Empty(t, gaugeValues(t))
}

func TestTranslator_DrainsTheBacklogThenPolls(t *testing.T) {
	t.Parallel()

	changes := mocks.NewMockChangesService(gomock.NewController(t))
	polled := make(chan struct{})

	gomock.InOrder(
		changes.EXPECT().Translate(gomock.Any()).Return(100, nil),
		changes.EXPECT().Translate(gomock.Any()).Return(3, nil),
		changes.EXPECT().Translate(gomock.Any()).Return(0, nil),
		changes.EXPECT().Translate(gomock.Any()).DoAndReturn(func(context.Context) (int, error) {
			close(polled)

			return 0, nil
		}),
		changes.EXPECT().Translate(gomock.Any()).Return(0, nil).AnyTimes(),
	)

	start(t, translator.New(changes, translator.WithPollInterval(10*time.Millisecond)))

	requireClosed(t, polled)
}

func TestTranslator_KeepsPollingAfterAFailedStep(t *testing.T) {
	t.Parallel()

	changes := mocks.NewMockChangesService(gomock.NewController(t))
	retried := make(chan struct{})

	gomock.InOrder(
		changes.EXPECT().Translate(gomock.Any()).Return(0, errors.ErrMock),
		changes.EXPECT().Translate(gomock.Any()).DoAndReturn(func(context.Context) (int, error) {
			close(retried)

			return 0, nil
		}),
		changes.EXPECT().Translate(gomock.Any()).Return(0, nil).AnyTimes(),
	)

	start(t, translator.New(changes, translator.WithPollInterval(10*time.Millisecond)))

	requireClosed(t, retried)
}

func TestTranslator_StopWaitsForTheRunningStep(t *testing.T) {
	t.Parallel()

	changes := mocks.NewMockChangesService(gomock.NewController(t))
	running := make(chan struct{})
	finished := make(chan struct{})

	changes.EXPECT().Translate(gomock.Any()).DoAndReturn(func(ctx context.Context) (int, error) {
		close(running)
		<-ctx.Done()
		close(finished)

		return 0, ctx.Err()
	})

	worker := translator.New(changes)
	require.NoError(t, worker.Start(t.Context()))
	requireClosed(t, running)

	require.NoError(t, worker.Stop(t.Context()))
	requireClosed(t, finished)
}

func start(t *testing.T, worker *translator.Translator) {
	t.Helper()

	require.NoError(t, worker.Start(t.Context()))
	t.Cleanup(func() { require.NoError(t, worker.Stop(context.Background())) })
}

func requireClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(waitFor):
		t.Fatal("timed out")
	}
}

func gaugeValues(t *testing.T) []int64 {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, metrics.Collect(t.Context(), &rm))

	var values []int64

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != oldestPendingAge {
				continue
			}

			gauge, ok := m.Data.(metricdata.Gauge[int64])
			require.True(t, ok, "%s is a %T", oldestPendingAge, m.Data)

			for _, dp := range gauge.DataPoints {
				values = append(values, dp.Value)
			}
		}
	}

	return values
}
