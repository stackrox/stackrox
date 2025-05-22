package vulnerabilities

import (
	"context"
	"iter"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var getters = []common.LabelGetter[*finding]{
	{Label: "Cluster", Getter: func(f *finding) string { return f.deployment.GetClusterName() }},
	{Label: "Namespace", Getter: func(f *finding) string { return f.deployment.GetNamespace() }},
	{Label: "Deployment", Getter: func(f *finding) string { return f.deployment.GetName() }},
	{Label: "ImageID", Getter: func(f *finding) string { return f.image.GetId() }},
	{Label: "ImageRegistry", Getter: func(f *finding) string { return f.name.GetRegistry() }},
	{Label: "ImageRemote", Getter: func(f *finding) string { return f.name.GetRemote() }},
	{Label: "ImageTag", Getter: func(f *finding) string { return f.name.GetTag() }},
	{Label: "Component", Getter: func(f *finding) string { return f.component.GetName() }},
	{Label: "ComponentVersion", Getter: func(f *finding) string { return f.component.GetVersion() }},
	{Label: "CVE", Getter: func(f *finding) string { return f.vuln.GetCve() }},
	{Label: "CVSS", Getter: func(f *finding) string { return strconv.FormatFloat(float64(f.vuln.GetCvss()), 'f', 1, 32) }},
	{Label: "OperatingSystem", Getter: func(f *finding) string { return f.image.GetScan().GetOperatingSystem() }},
	{Label: "Severity", Getter: func(f *finding) string { return f.vuln.GetSeverity().String() }},
	{Label: "SeverityV2", Getter: func(f *finding) string { return f.vuln.GetCvssV2().GetSeverity().String() }},
	{Label: "SeverityV3", Getter: func(f *finding) string { return f.vuln.GetCvssV3().GetSeverity().String() }},
	{Label: "IsFixable", Getter: isFixable},
}

type finding struct {
	deployment *storage.Deployment
	image      *storage.Image
	name       *storage.ImageName
	component  *storage.EmbeddedImageScanComponent
	vuln       *storage.EmbeddedVulnerability
}

func MakeTrackerConfig(gauge func(string, prometheus.Labels, int)) *common.TrackerConfig[*finding] {
	tc := common.MakeTrackerConfig(
		"vulnerabilities",
		"aggregated CVEs",
		getters,
		common.Bind3rd(trackVulnerabilityMetrics, deploymentDS.Singleton()),
		gauge)
	return tc
}

func isFixable(f *finding) string {
	if f.vuln.GetFixedBy() == "" {
		return "false"
	}
	return "true"
}

func trackVulnerabilityMetrics(ctx context.Context, mle common.MetricLabelsExpressions, ds deploymentDS.DataStore) iter.Seq[*finding] {
	// Check if image data is needed:
	trackImageData := false
mleLoop:
	for _, expr := range mle {
		for label := range expr {
			if label != "Cluster" && label != "Namespace" && label != "Deployment" {
				trackImageData = true
				break mleLoop
			}
		}
	}
	return func(yield func(*finding) bool) {
		finding := &finding{}
		// Optimization opportunity:
		// The resource filter (mle) is known at this point, so a more precise
		// query could be constructed here.
		_ = ds.WalkByQuery(ctx, search.EmptyQuery(), func(deployment *storage.Deployment) error {
			finding.deployment = deployment
			if trackImageData {
				images, err := ds.GetImagesForDeployment(ctx, deployment)
				if err != nil {
					return nil // Nothing can be done with this error here.
				}
				if !forEachFinding(yield, finding, images) {
					return common.ErrStopIterator
				}
			} else {
				if !yield(finding) {
					return common.ErrStopIterator
				}
			}
			return nil
		})
	}
}

func forEachFinding(yield func(*finding) bool, f *finding, images []*storage.Image) bool {
	for _, f.image = range images {
		for _, f.component = range f.image.GetScan().GetComponents() {
			for _, f.vuln = range f.component.GetVulns() {
				for _, f.name = range f.image.GetNames() {
					if !yield(f) {
						return false
					}
				}
			}
		}
	}
	return true
}
