package changes_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/mock/gomock"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/lcnascimento/go-kit/errors"

	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/application/changes"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/application/internal/mocks"
	"github.com/lcnascimento/boilerplate/backend/services/iam/internal/idomain"
	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"
)

var metrics = sdkmetric.NewManualReader()

func TestMain(m *testing.M) {
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(metrics)))

	os.Exit(m.Run())
}

func TestTranslate_TranslatesABatchOfOneHundred(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockChangesRepository(gomock.NewController(t))
	repo.EXPECT().Translate(gomock.Any(), int32(100)).Return([]*idomain.Event{
		{Message: &events.UserCreated{}},
		{Message: &events.UserDeleted{}},
	}, nil)

	translated, err := changes.New(repo).Translate(t.Context())

	require.NoError(t, err)
	require.Equal(t, 2, translated)
}

func TestTranslate_CountsTheTranslatedChangesByEventType(t *testing.T) {
	repo := mocks.NewMockChangesRepository(gomock.NewController(t))
	repo.EXPECT().Translate(gomock.Any(), gomock.Any()).Return([]*idomain.Event{
		{Message: &events.PasswordChanged{}},
		{Message: &events.PasswordChanged{}},
		{Message: &events.UserDeleted{}},
	}, nil)

	before := translatedByType(t)

	_, err := changes.New(repo).Translate(t.Context())
	require.NoError(t, err)

	after := translatedByType(t)
	require.Equal(t, int64(2), after["boilerplate.iam.v1.PasswordChanged"]-before["boilerplate.iam.v1.PasswordChanged"])
	require.Equal(t, int64(1), after["boilerplate.iam.v1.UserDeleted"]-before["boilerplate.iam.v1.UserDeleted"])
}

func TestTranslate_WrapsRepositoryErrors(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockChangesRepository(gomock.NewController(t))
	repo.EXPECT().Translate(gomock.Any(), gomock.Any()).Return(nil, errors.ErrMock)

	_, err := changes.New(repo).Translate(t.Context())

	require.ErrorIs(t, err, changes.ErrTranslateChanges)
	require.ErrorIs(t, err, errors.ErrMock)
}

func TestOldestPendingAge_IsTheRepositoryAge(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockChangesRepository(gomock.NewController(t))
	repo.EXPECT().OldestPendingAge(gomock.Any()).Return(90*time.Second, nil)

	age, err := changes.New(repo).OldestPendingAge(t.Context())

	require.NoError(t, err)
	require.Equal(t, 90*time.Second, age)
}

func TestOldestPendingAge_WrapsRepositoryErrors(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockChangesRepository(gomock.NewController(t))
	repo.EXPECT().OldestPendingAge(gomock.Any()).Return(time.Duration(0), errors.ErrMock)

	_, err := changes.New(repo).OldestPendingAge(t.Context())

	require.ErrorIs(t, err, changes.ErrOldestPendingAge)
	require.ErrorIs(t, err, errors.ErrMock)
}

func translatedByType(t *testing.T) map[string]int64 {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, metrics.Collect(t.Context(), &rm))

	counts := map[string]int64{}

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "iam.identity_change.translated" {
				continue
			}

			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "iam.identity_change.translated is a %T", m.Data)

			for _, dp := range sum.DataPoints {
				eventType, _ := dp.Attributes.Value("cloudevents.event_type")
				counts[eventType.AsString()] = dp.Value
			}
		}
	}

	return counts
}
