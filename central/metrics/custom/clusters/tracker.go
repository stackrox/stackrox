package clusters

import (
	"context"
	"iter"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
)

func New(ds clusterDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"health",
		"clusters",
		LazyLabels,
		func(ctx context.Context, _ tracker.MetricDescriptors) iter.Seq[*finding] {
			return trackClusters(ctx, ds)
		},
	)
}

func trackClusters(ctx context.Context, ds clusterDS.DataStore) iter.Seq[*finding] {
	f := finding{}
	return func(yield func(*finding) bool) {
		if ds == nil {
			return
		}
		_ = ds.WalkClusters(ctx, func(cluster *storage.Cluster) error {
			f.cluster = cluster
			if !yield(&f) {
				return tracker.ErrStopIterator
			}
			return nil
		})
	}
}
