package forwarder_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/mock/gomock"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/internal/mocks"
	"github.com/lcnascimento/boilerplate/backend/services/notes/internal/transport/worker/forwarder"
)

const oldestPendingAge = "notes.outbox.oldest_pending.age"

func TestJanitor_PurgesOnStart(t *testing.T) {
	t.Parallel()

	outbox := mocks.NewMockOutboxService(gomock.NewController(t))
	purged := make(chan struct{})

	outbox.EXPECT().Purge(gomock.Any()).DoAndReturn(func(context.Context) error {
		close(purged)

		return nil
	})

	janitor := forwarder.NewJanitor(outbox)
	require.NoError(t, janitor.Start(t.Context()))
	t.Cleanup(func() { require.NoError(t, janitor.Stop(context.Background())) })

	requireClosed(t, purged)
}

func TestJanitor_KeepsPurgingAfterAFailure(t *testing.T) {
	t.Parallel()

	outbox := mocks.NewMockOutboxService(gomock.NewController(t))
	recovered := make(chan struct{})

	gomock.InOrder(
		outbox.EXPECT().Purge(gomock.Any()).Return(errors.ErrMock),
		outbox.EXPECT().Purge(gomock.Any()).DoAndReturn(func(context.Context) error {
			close(recovered)

			return nil
		}),
		outbox.EXPECT().Purge(gomock.Any()).Return(nil).AnyTimes(),
	)

	janitor := forwarder.NewJanitorPurgingEvery(outbox, time.Millisecond)
	require.NoError(t, janitor.Start(t.Context()))
	t.Cleanup(func() { require.NoError(t, janitor.Stop(context.Background())) })

	requireClosed(t, recovered)
}

func TestJanitor_StopsPurging(t *testing.T) {
	t.Parallel()

	outbox := mocks.NewMockOutboxService(gomock.NewController(t))

	var calls atomic.Int64

	outbox.EXPECT().Purge(gomock.Any()).DoAndReturn(func(context.Context) error {
		calls.Add(1)

		return nil
	}).MinTimes(1)

	janitor := forwarder.NewJanitorPurgingEvery(outbox, time.Millisecond)
	require.NoError(t, janitor.Start(t.Context()))
	require.Eventually(t, func() bool { return calls.Load() > 0 }, waitFor, time.Millisecond)
	require.NoError(t, janitor.Stop(t.Context()))

	stoppedAt := calls.Load()

	time.Sleep(20 * time.Millisecond)

	require.Equal(t, stoppedAt, calls.Load())
}

func TestJanitor_ReportsTheOldestPendingAgeUntilStopped(t *testing.T) {
	outbox := mocks.NewMockOutboxService(gomock.NewController(t))
	outbox.EXPECT().Purge(gomock.Any()).Return(nil).AnyTimes()
	outbox.EXPECT().OldestPendingAge(gomock.Any()).Return(90*time.Second, nil)

	janitor := forwarder.NewJanitor(outbox)
	require.NoError(t, janitor.Start(t.Context()))

	got, ok := gauge(t, oldestPendingAge)
	require.True(t, ok)
	require.Equal(t, int64(90), got)

	require.NoError(t, janitor.Stop(t.Context()))

	_, ok = gauge(t, oldestPendingAge)
	require.False(t, ok)
}

func TestJanitor_SkipsTheOldestPendingAgeWhenItCannotReadIt(t *testing.T) {
	outbox := mocks.NewMockOutboxService(gomock.NewController(t))
	outbox.EXPECT().Purge(gomock.Any()).Return(nil).AnyTimes()
	outbox.EXPECT().OldestPendingAge(gomock.Any()).Return(time.Duration(0), errors.ErrMock)

	janitor := forwarder.NewJanitor(outbox)
	require.NoError(t, janitor.Start(t.Context()))
	t.Cleanup(func() { require.NoError(t, janitor.Stop(context.Background())) })

	_, ok := gauge(t, oldestPendingAge)
	require.False(t, ok)
}

func requireClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(waitFor):
		t.Fatal("timed out")
	}
}

func gauge(t *testing.T, name string) (int64, bool) {
	t.Helper()

	var collected metricdata.ResourceMetrics
	require.NoError(t, metrics.Collect(t.Context(), &collected))

	for _, scope := range collected.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != name {
				continue
			}

			data, ok := m.Data.(metricdata.Gauge[int64])
			require.True(t, ok, "%s is a %T", name, m.Data)

			if len(data.DataPoints) == 0 {
				return 0, false
			}

			return data.DataPoints[0].Value, true
		}
	}

	return 0, false
}
