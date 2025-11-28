package image_vulnerabilities

import (
	"context"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/central/views/deploymentcve"
)

func New(view deploymentcve.CveView) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"image_vuln",
		"image vulnerabilities",
		lazyLabels,
		func(ctx context.Context, _ tracker.MetricDescriptors) tracker.FindingErrorSequence[*finding] {
			return track(ctx, view)
		},
	)
}

func track(ctx context.Context, view deploymentcve.CveView) tracker.FindingErrorSequence[*finding] {
	return func(yield func(*finding, error) bool) {
		collector := tracker.NewFindingCollector(yield)
		collector.Finally(view.WalkVulnFindings(ctx, collector.Yield))
	}
}
