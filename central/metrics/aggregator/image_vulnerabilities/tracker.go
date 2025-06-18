package image_vulnerabilities

import (
	"context"
	"iter"

	"github.com/prometheus/client_golang/prometheus"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

func New(gauge func(string, prometheus.Labels, int), ds deploymentDS.DataStore) *common.TrackerBase[finding] {
	return common.MakeTrackerBase(
		"vulnerabilities",
		"aggregated CVEs",
		lazyLabels,
		func(ctx context.Context, mcfg common.MetricsConfiguration) iter.Seq[*finding] {
			return trackVulnerabilityMetrics(ctx, mcfg, ds)
		},
		gauge)
}

func trackVulnerabilityMetrics(ctx context.Context, _ common.MetricsConfiguration, ds deploymentDS.DataStore) iter.Seq[*finding] {
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
					return common.ErrStopIterator
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
