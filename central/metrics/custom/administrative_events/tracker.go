package administrative_events

import (
	"context"

	adminEventDS "github.com/stackrox/rox/central/administration/events/datastore"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/pkg/search"
)

func New(ds adminEventDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeGlobalTrackerBase(
		"admin_event",
		"administrative events",
		LazyLabels,
		func(ctx context.Context, _ tracker.MetricDescriptors) tracker.FindingErrorSequence[*finding] {
			return track(ctx, ds)
		},
	)
}

func track(ctx context.Context, ds adminEventDS.DataStore) tracker.FindingErrorSequence[*finding] {
	return func(yield func(*finding, error) bool) {
		if ds == nil {
			return
		}
		collector := tracker.NewFindingCollector(yield)
		events, err := ds.ListEvents(ctx, search.EmptyQuery())
		if err != nil {
			collector.Error(err)
			return
		}
		for _, event := range events {
			if err := collector.Yield(&finding{event}); err != nil {
				return
			}
		}
	}
}
