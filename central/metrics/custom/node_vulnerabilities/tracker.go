package node_vulnerabilities

import (
	"context"
	"iter"

	cveDS "github.com/stackrox/rox/central/cve/node/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	nodeDS "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type datastores struct {
	nDS   nodeDS.DataStore
	cveDS cveDS.DataStore
}

func New(registryFactory func(string) metrics.CustomRegistry, nds nodeDS.DataStore, cveds cveDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"node vulnerabilities",
		"node CVEs",
		lazyLabels,
		func(ctx context.Context, md tracker.MetricDescriptors) iter.Seq[*finding] {
			return trackVulnerabilityMetrics(ctx, md, datastores{nds, cveds})
		},
		registryFactory)
}

func trackVulnerabilityMetrics(ctx context.Context, md tracker.MetricDescriptors, ds datastores) iter.Seq[*finding] {
	f := finding{}
	return func(yield func(*finding) bool) {
		_ = ds.nDS.WalkByQuery(ctx, search.EmptyQuery(), func(node *storage.Node) error {
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
