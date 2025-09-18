package policies

import (
	"context"
	"iter"
	"strconv"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	policyDS "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/pkg/search"
)

var lazyLabels = []tracker.LazyLabel[*finding]{
	{Label: "Enabled", Getter: func(f *finding) string {
		return strconv.FormatBool(f.enabled)
	}},
}

type finding struct {
	tracker.CommonFinding
	enabled bool
	n       int
	err     error
}

func GetLabels() []string {
	result := make([]string, 0, len(lazyLabels))
	for _, l := range lazyLabels {
		result = append(result, string(l.Label))
	}
	return result
}

func New(ds policyDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"health",
		"health",
		lazyLabels,
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
