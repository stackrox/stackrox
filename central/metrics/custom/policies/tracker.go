package policies

import (
	"context"
	"iter"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	policyDS "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/pkg/search"
)

func New(ds policyDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"health",
		"health",
		LazyLabels,
		func(ctx context.Context, _ tracker.MetricDescriptors) iter.Seq[*finding] {
			return track(ctx, ds)
		},
	)
}

func track(ctx context.Context, ds policyDS.DataStore) iter.Seq[*finding] {
	f := finding{}
	return func(yield func(*finding) bool) {
		if ds == nil {
			return
		}
		qb := search.NewQueryBuilder()
		qb.AddBools("Disabled", false)
		f.enabled = true
		f.n, f.err = ds.Count(ctx, qb.ProtoQuery())
		if !yield(&f) {
			return
		}
		qb = search.NewQueryBuilder()
		qb.AddBools("Disabled", true)
		f.enabled = false
		f.n, f.err = ds.Count(ctx, qb.ProtoQuery())
		if !yield(&f) {
			return
		}
	}
}
