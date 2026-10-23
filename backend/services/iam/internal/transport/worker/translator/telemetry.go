package translator

import (
	"context"

	"go.opentelemetry.io/otel"

	otelmetric "go.opentelemetry.io/otel/metric"
)

var (
	pkg   = "iam/transport/worker/translator"
	meter = otel.Meter(pkg)
)

func (t *Translator) registerOldestPendingAge() (otelmetric.Registration, error) {
	gauge, err := meter.Int64ObservableGauge("iam.identity_change.oldest_pending.age",
		otelmetric.WithDescription("Age of the oldest identity change not yet translated into an Integration Event, 0 when none"),
		otelmetric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	return meter.RegisterCallback(func(ctx context.Context, observer otelmetric.Observer) error {
		if age, err := t.changes.OldestPendingAge(ctx); err == nil {
			observer.ObserveInt64(gauge, int64(age.Seconds()))
		}

		return nil
	}, gauge)
}
