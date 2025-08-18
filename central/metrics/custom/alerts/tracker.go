package alerts

import (
	"context"
	"iter"

	alertDS "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

func New(registry metrics.CustomRegistry, ds alertDS.DataStore) *tracker.TrackerBase[finding] {
	return tracker.MakeTrackerBase(
		"alerts",
		"policy violation alerts",
		lazyLabels,
		func(ctx context.Context, _ tracker.MetricsConfiguration) iter.Seq[*finding] {
			return trackAlertsMetrics(ctx, ds)
		},
		registry)
}

func trackAlertsMetrics(ctx context.Context, ds alertDS.DataStore) iter.Seq[*finding] {
	f := finding{}
	return func(yield func(*finding) bool) {
		_ = ds.WalkByQuery(ctx, search.EmptyQuery(), func(a *storage.Alert) error {
			f.Alert = a
			for _, v := range a.GetViolations() {
				f.Alert_Violation = v
				if !yield(&f) {
					return tracker.ErrStopIterator
				}
			}
			return nil
		})
	}
}
