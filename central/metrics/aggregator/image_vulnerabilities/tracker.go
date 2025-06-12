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

var getters = []common.LabelGetter[*finding]{
	{Label: "Cluster", Getter: func(f *finding) string { return f.deployment.GetClusterName() }},
	{Label: "Namespace", Getter: func(f *finding) string { return f.deployment.GetNamespace() }},
	{Label: "Deployment", Getter: func(f *finding) string { return f.deployment.GetName() }},
	{Label: "IsPlatformWorkload", Getter: isPlatformWorkload},

	{Label: "ImageID", Getter: func(f *finding) string { return f.image.GetId() }},
	{Label: "ImageRegistry", Getter: func(f *finding) string { return f.name.GetRegistry() }},
	{Label: "ImageRemote", Getter: func(f *finding) string { return f.name.GetRemote() }},
	{Label: "ImageTag", Getter: func(f *finding) string { return f.name.GetTag() }},
	{Label: "Component", Getter: func(f *finding) string { return f.component.GetName() }},
	{Label: "ComponentVersion", Getter: func(f *finding) string { return f.component.GetVersion() }},
	{Label: "OperatingSystem", Getter: func(f *finding) string { return f.image.GetScan().GetOperatingSystem() }},

	{Label: "CVE", Getter: func(f *finding) string { return f.vuln.GetCve() }},
	{Label: "CVSS", Getter: func(f *finding) string { return strconv.FormatFloat(float64(f.vuln.GetCvss()), 'f', 1, 32) }},
	{Label: "Severity", Getter: func(f *finding) string { return f.vuln.GetSeverity().String() }},
	{Label: "SeverityV2", Getter: func(f *finding) string { return f.vuln.GetCvssV2().GetSeverity().String() }},
	{Label: "SeverityV3", Getter: func(f *finding) string { return f.vuln.GetCvssV3().GetSeverity().String() }},
	{Label: "IsFixable", Getter: func(f *finding) string { return strconv.FormatBool(f.vuln.GetFixedBy() != "") }},
}

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

func MakeTrackerConfig(gauge func(string, prometheus.Labels, int), ds deploymentDS.DataStore) *common.TrackerConfig[*finding] {
	tc := common.MakeTrackerConfig(
		"vulnerabilities",
		"aggregated CVEs",
		getters,
		common.Bind4th(trackVulnerabilityMetrics, ds),
		gauge)
	return tc
}

func trackVulnerabilityMetrics(ctx context.Context, query *v1.Query, mcfg common.MetricsConfiguration, ds deploymentDS.DataStore) iter.Seq[*finding] {
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
