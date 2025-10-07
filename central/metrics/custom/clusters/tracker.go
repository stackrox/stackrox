package clusters

import (
	"context"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
)

func New(ds clusterDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"health",
		"clusters",
		LazyLabels,
		func(ctx context.Context, _ tracker.MetricDescriptors) tracker.FindingErrorSequence[*finding] {
			return track(ctx, ds)
		},
	)
}

func track(ctx context.Context, ds clusterDS.DataStore) tracker.FindingErrorSequence[*finding] {
	return func(yield func(*finding, error) bool) {
		if ds == nil {
			return
		}
		var f finding
		collector := tracker.NewFindingCollector(yield)
		collector.Finally(ds.WalkClusters(ctx, func(cluster *storage.Cluster) error {
			f.Cluster = cluster
			return collector.Yield(&f)
		}))
	}
}
