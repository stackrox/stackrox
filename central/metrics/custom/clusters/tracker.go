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
			return track(ctx, ds)
		},
	)
}

func track(ctx context.Context, ds clusterDS.DataStore) iter.Seq[*finding] {
	return func(yield func(*finding) bool) {
		if ds == nil {
			return
		}
		var f finding
		collect := tracker.NewFindingCollector(yield)
		defer collect.Finally(&f)
		f.SetError(ds.WalkClusters(ctx, func(cluster *storage.Cluster) error {
			f.Cluster = cluster
			return collect(&f)
		}))
	}
}
