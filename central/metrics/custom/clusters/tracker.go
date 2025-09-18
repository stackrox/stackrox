package clusters

import (
	"context"
	"iter"
	"strconv"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

var lazyLabels = []tracker.LazyLabel[*finding]{
	{Label: "Cluster", Getter: func(f *finding) string { return f.cluster }},
	{Label: "IsHealthy", Getter: func(f *finding) string { return strconv.FormatBool(f.healthy) }},
}

type finding struct {
	tracker.CommonFinding
	cluster string
	healthy bool
	n       int
	err     error
}

func New(ds clusterDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"clusters",
		"clusters",
		lazyLabels,
		func(ctx context.Context, _ tracker.MetricDescriptors) iter.Seq[*finding] {
			return trackClusters(ctx, ds)
		},
	)
}

func trackClusters(ctx context.Context, ds clusterDS.DataStore) iter.Seq[*finding] {
	f := finding{}
	return func(yield func(*finding) bool) {
		_ = ds.WalkClusters(ctx, func(obj *storage.Cluster) error {
			if !yield(&f) {
				return errox.ResourceExhausted
			}
			return nil
		})
	}
}
