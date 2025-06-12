package image_vulnerabilities

import (
	"context"
	"iter"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/central/platform/matcher"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

type finding struct {
	common.OneOrMore
	deployment *storage.Deployment
	image      *storage.Image
	name       *storage.ImageName
	component  *storage.EmbeddedImageScanComponent
	vuln       *storage.EmbeddedVulnerability
}

func isPlatformWorkload(f *finding) string {
	p, _ := matcher.Singleton().MatchDeployment(f.deployment)
	return strconv.FormatBool(p)
}

func New(gauge func(string, prometheus.Labels, int), ds deploymentDS.DataStore) *common.TrackerBase[*finding] {
	tc := common.MakeTrackerBase(
		"vulnerabilities",
		"aggregated CVEs",
		getters,
		func(ctx context.Context, q *v1.Query, mcfg common.MetricsConfiguration) iter.Seq[*finding] {
			return trackVulnerabilityMetrics(ctx, q, mcfg, ds)
		},
		gauge)
	return tc
}

func trackVulnerabilityMetrics(ctx context.Context, query *v1.Query, _ common.MetricsConfiguration, ds deploymentDS.DataStore) iter.Seq[*finding] {
	return func(yield func(*finding) bool) {
		finding := &finding{}
		_ = ds.WalkByQuery(ctx, query, func(deployment *storage.Deployment) error {
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
