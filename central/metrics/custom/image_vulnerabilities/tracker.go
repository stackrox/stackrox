package image_vulnerabilities

import (
	"context"
	"iter"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

func New(ds deploymentDS.DataStore) *tracker.TrackerBase[*finding] {
	return tracker.MakeTrackerBase(
		"vulnerabilities",
		"CVEs",
		lazyLabels,
		func(ctx context.Context, md tracker.MetricDescriptors) iter.Seq[*finding] {
			return trackVulnerabilityMetrics(ctx, md, ds)
		},
	)
}

func trackVulnerabilityMetrics(ctx context.Context, _ tracker.MetricDescriptors, ds deploymentDS.DataStore) iter.Seq[*finding] {
	ctx = sac.WithGlobalAccessScopeChecker(ctx, sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(resources.Deployment, resources.Image)))

	return func(yield func(*finding) bool) {
		finding := &finding{}
		_ = ds.WalkByQuery(ctx, search.EmptyQuery(), func(deployment *storage.Deployment) error {
			finding.deployment = deployment
			images, err := ds.GetImagesForDeployment(ctx, deployment)
			if err != nil {
				return nil // Nothing can be done with this error here.
			}
			for _, finding.image = range images {
				if !forEachImageVuln(yield, finding) {
					return tracker.ErrStopIterator
				}
			}
			return nil
		})
	}
}

// forEachImageVuln yields a finding for every vulnerability associated with
// each image name.
func forEachImageVuln(yield func(*finding) bool, f *finding) bool {
	for _, f.component = range f.image.GetScan().GetComponents() {
		for _, f.vuln = range f.component.GetVulns() {
			for _, f.name = range f.image.GetNames() {
				if !yield(f) {
					return false
				}
			}
		}
	}
	return true
}
