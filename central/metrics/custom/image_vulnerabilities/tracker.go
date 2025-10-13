package image_vulnerabilities

import (
	"context"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

func New(ds deploymentDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"image_vuln",
		"image vulnerabilities",
		lazyLabels,
		func(ctx context.Context, md tracker.MetricDescriptors) tracker.FindingErrorSequence[*finding] {
			return track(ctx, ds)
		},
	)
}

func track(ctx context.Context, ds deploymentDS.DataStore) tracker.FindingErrorSequence[*finding] {
	return func(yield func(*finding, error) bool) {
		var f finding
		collector := tracker.NewFindingCollector(yield)
		collector.Finally(ds.WalkByQuery(ctx, search.EmptyQuery(), func(deployment *storage.Deployment) error {
			f.deployment = deployment
			images, err := ds.GetImagesForDeployment(ctx, deployment)
			if err != nil {
				return err
			}
			for _, f.image = range images {
				if err := forEachImageVuln(collector, &f); err != nil {
					return err
				}
			}
			return nil
		}))
	}
}

// forEachImageVuln yields a finding for every vulnerability associated with
// each image name.
func forEachImageVuln(collector tracker.Collector[*finding], f *finding) error {
	for _, f.component = range f.image.GetScan().GetComponents() {
		for _, f.vuln = range f.component.GetVulns() {
			for _, f.name = range f.image.GetNames() {
				if err := collector.Yield(f); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
