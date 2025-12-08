package node_vulnerabilities

import (
	"context"

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
		func(ctx context.Context, _ tracker.MetricDescriptors) tracker.FindingErrorSequence[*finding] {
			return track(ctx, nodes)
		})
}

func track(ctx context.Context, ds nodeDS.DataStore) tracker.FindingErrorSequence[*finding] {
	return func(yield func(*finding, error) bool) {
		if ds == nil {
			return
		}
		var f finding
		collector := tracker.NewFindingCollector(yield)
		collector.Finally(ds.WalkByQuery(ctx, search.EmptyQuery(), func(node *storage.Node) error {
			f.node = node
			return forEachNode(collector, &f)
		}))
	}
}

func forEachNode(collector tracker.Collector[*finding], f *finding) error {
	for _, f.component = range f.node.GetScan().GetComponents() {
		for _, f.vulnerability = range f.component.GetVulnerabilities() {
			if err := collector.Yield(f); err != nil {
				return err
			}
		}
	}
	return nil
}
