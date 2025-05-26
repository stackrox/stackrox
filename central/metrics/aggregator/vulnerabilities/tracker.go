package vulnerabilities

import (
	"context"
	"iter"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
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

type datastores struct {
	dDS deploymentDS.DataStore
	iDS imageDS.DataStore
}

func MakeTrackerConfig(gauge func(string, prometheus.Labels, int)) *common.TrackerConfig[*finding] {
	tc := common.MakeTrackerConfig(
		"vulnerabilities",
		"aggregated CVEs",
		getters,
		common.Bind3rd(trackVulnerabilityMetrics, datastores{deploymentDS.Singleton(), imageDS.Singleton()}),
		gauge)
	return tc
}

func isFixable(f *finding) string {
	if f.vuln.GetFixedBy() == "" {
		return "false"
	}
	return "true"
}

func trackVulnerabilityMetrics(ctx context.Context, mle common.MetricLabelsExpressions, ds datastores) iter.Seq[*finding] {
	// Optimization opportunity:
	// The resource filter (mle) is known at this point, so a more precise
	// query could be constructed here.
	query := search.EmptyQuery()

	return func(yield func(*finding) bool) {
		finding := &finding{}

		if queryDeploymentData(mle) {
			_ = ds.dDS.WalkByQuery(ctx, query, func(deployment *storage.Deployment) error {
				finding.deployment = deployment
				images, err := ds.dDS.GetImagesForDeployment(ctx, deployment)
				if err != nil {
					return nil // Nothing can be done with this error here.
				}
				for _, finding.image = range images {
					if !forEachFinding(yield, finding) {
						return common.ErrStopIterator
					}
				}
				return nil
			})
		} else {
			// Optimization: do not query deployments.
			_ = ds.iDS.WalkByQuery(ctx, query, func(image *storage.Image) error {
				finding.image = image
				if !forEachFinding(yield, finding) {
					return common.ErrStopIterator
				}
				return nil
			})
		}
	}
}

func queryDeploymentData(mle common.MetricLabelsExpressions) bool {
	for _, expr := range mle {
		for label := range expr {
			if label == "Cluster" || label == "Namespace" || label == "Deployment" {
				return true
			}
		}
	}
	return false
}

func forEachFinding(yield func(*finding) bool, f *finding) bool {
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
