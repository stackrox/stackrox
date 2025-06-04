package node_vulnerabilities

import (
	"context"
	"iter"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	nodeDS "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

func New(nodes nodeDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"node_vuln",
		"node CVEs",
		lazyLabels,
		func(ctx context.Context, _ tracker.MetricDescriptors) iter.Seq[*finding] {
			return track(ctx, nodes)
		})
}

func track(ctx context.Context, ds nodeDS.DataStore) iter.Seq[*finding] {
	return func(yield func(*finding) bool) {
		var f finding
		f.err = ds.WalkByQuery(ctx, search.EmptyQuery(), func(node *storage.Node) error {
			f.node = node
			if !forEachNode(yield, &f) {
				return tracker.ErrStopIterator
			}
			return nil
		})
		// Report walking error.
		if f.err != nil && !errors.Is(f.err, tracker.ErrStopIterator) {
			yield(&f)
		}
	}
}

func forEachNode(yield func(*finding) bool, f *finding) bool {
	for _, f.component = range f.node.GetScan().GetComponents() {
		for _, f.vulnerability = range f.component.GetVulnerabilities() {
			if !yield(f) {
				return false
			}
		}
	}
	return true
}
