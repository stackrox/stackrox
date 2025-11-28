package policy_violations

import (
	"context"

	alertDS "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var onlyActiveViolations = search.NewQueryBuilder().
	AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String()).
	ProtoQuery()

func New(ds alertDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"policy_violation",
		"policy violations",
		lazyLabels,
		func(ctx context.Context, cfg *tracker.Configuration) tracker.FindingErrorSequence[*finding] {
			var query *v1.Query
			if cfg.AllMetricsHaveFilter(tracker.Label("State"), storage.ViolationState_ACTIVE.String()) {
				query = onlyActiveViolations
			} else {
				query = search.EmptyQuery()
			}
			return track(ctx, ds, query)
		},
	)
}

func track(ctx context.Context, ds alertDS.DataStore, query *v1.Query) tracker.FindingErrorSequence[*finding] {
	return func(yield func(*finding, error) bool) {
		var f finding
		collector := tracker.NewFindingCollector(yield)
		collector.Finally(ds.WalkByQuery(ctx, query, func(a *storage.Alert) error {
			f.Alert = a
			for _, v := range a.GetViolations() {
				f.Alert_Violation = v
				if err := collector.Yield(&f); err != nil {
					return err
				}
			}
			return nil
		}))
	}
}
