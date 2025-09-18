package total_enabled_policies

import (
	"context"
	"iter"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	policyDS "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/pkg/search"
)

var lazyLabels = []tracker.LazyLabel[*finding]{}

type finding struct {
	tracker.CommonFinding
	n   int
	err error
}

func New(ds policyDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"health",
		"health",
		lazyLabels,
		func(ctx context.Context, _ tracker.MetricDescriptors) iter.Seq[*finding] {
			return trackPolicies(ctx, ds)
		},
	)
}

func trackPolicies(ctx context.Context, ds policyDS.DataStore) iter.Seq[*finding] {
	qb := search.NewQueryBuilder()
	qb.AddBools("Disabled", false)
	f := finding{}
	return func(yield func(*finding) bool) {
		f.n, f.err = ds.Count(ctx, qb.ProtoQuery())
		if !yield(&f) {
			return
		}
	}
}
