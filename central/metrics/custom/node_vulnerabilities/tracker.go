package node_vulnerabilities

import (
	"context"
	"iter"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	nodeDS "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

func New(nodes nodeDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"node vulnerabilities",
		"node CVEs",
		lazyLabels,
		func(ctx context.Context, md tracker.MetricDescriptors) iter.Seq[*finding] {
			return trackVulnerabilityMetrics(ctx, md, nodes)
		})
}

func trackVulnerabilityMetrics(ctx context.Context, md tracker.MetricDescriptors, ds nodeDS.DataStore) iter.Seq[*finding] {
	f := finding{}
	return func(yield func(*finding) bool) {
		_ = ds.WalkByQuery(ctx, search.EmptyQuery(), func(node *storage.Node) error {
			f.node = node
			if !forEachFinding(yield, &f) {
				return tracker.ErrStopIterator
			}
			return nil
		})
	}
}

func forEachFinding(yield func(*finding) bool, f *finding) bool {
	for _, f.component = range f.node.GetScan().GetComponents() {
		for _, f.vuln = range f.component.GetVulns() {
			if !yield(f) {
				return false
			}
		}
	}
	return true
}
