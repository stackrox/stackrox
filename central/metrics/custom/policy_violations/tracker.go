package policy_violations

import (
	"context"
	"iter"

	alertDS "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

func New(ds alertDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"policy_violation",
		"policy violations",
		lazyLabels,
		func(ctx context.Context, _ tracker.MetricDescriptors) iter.Seq[*finding] {
			return track(ctx, ds)
		},
	)
}

func track(ctx context.Context, ds alertDS.DataStore) iter.Seq[*finding] {
	return func(yield func(*finding) bool) {
		var f finding
		collector := tracker.NewFindingCollector(yield)
		defer collector.Finally(&f)
		f.SetError(ds.WalkByQuery(ctx, search.EmptyQuery(), func(a *storage.Alert) error {
			f.Alert = a
			for _, v := range a.GetViolations() {
				f.Alert_Violation = v
				if err := collector(&f); err != nil {
					return err
				}
			}
			return nil
		}))
	}
}
