package image_vulnerabilities

import (
	"context"

	cveDS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
)

func New(ds cveDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"image_vuln",
		"image vulnerabilities",
		lazyLabels,
		func(ctx context.Context, _ *tracker.Configuration) tracker.FindingErrorSequence[*finding] {
			return track(ctx, ds)
		},
	)
}

func track(ctx context.Context, ds cveDS.DataStore) tracker.FindingErrorSequence[*finding] {
	return func(yield func(*finding, error) bool) {
		collector := tracker.NewFindingCollector(yield)
		collector.Finally(ds.WalkDeploymentVulnFindings(ctx, collector.Yield))
	}
}
