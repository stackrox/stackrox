package image_vulnerabilities

import (
	"context"
	"iter"
	"slices"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	imageCVEDS "github.com/stackrox/rox/central/cve/image/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

var (
	getters = []common.LabelGetter[*finding]{
		{Label: "Cluster", Getter: func(f *finding) string { return f.deployment.GetClusterName() }},
		{Label: "Namespace", Getter: func(f *finding) string { return f.deployment.GetNamespace() }},
		{Label: "Deployment", Getter: func(f *finding) string { return f.deployment.GetName() }},

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

	deploymentLabels = []common.Label{"Cluster", "Namespace", "Deployment"}
	imageLabels      = []common.Label{"ImageID", "ImageRegistry", "ImageRemote", "ImageTag", "Component", "ComponentVersion", "OperatingSystem"}
)

type finding struct {
	deployment *storage.Deployment
	image      *storage.Image
	name       *storage.ImageName
	component  *storage.EmbeddedImageScanComponent
	vuln       *storage.EmbeddedVulnerability

	count int
}

func (f *finding) Count() int {
	if f.count > 0 {
		return f.count
	}
	return 1
}

type datastores struct {
	deployments deploymentDS.DataStore
	images      imageDS.DataStore
	cves        imageCVEDS.DataStore
}

func MakeTrackerConfig(gauge func(string, prometheus.Labels, int)) *common.TrackerConfig[*finding] {
	tc := common.MakeTrackerConfig(
		"vulnerabilities",
		"aggregated CVEs",
		getters,
		common.Bind4th(trackVulnerabilityMetrics, datastores{
			deploymentDS.Singleton(),
			imageDS.Singleton(),
			imageCVEDS.Singleton(),
		}),
		gauge)
	return tc
}

func trackVulnerabilityMetrics(ctx context.Context, query *v1.Query, mle common.MetricLabelsExpressions, ds datastores) iter.Seq[*finding] {
	// Optimization opportunity:
	// The resource filter (mle) is known at this point, so a more precise
	// query could be constructed here.
	queryDeploymentData := queryDeploymentData(mle)
	queryImageData := !queryDeploymentData && queryImageData(mle)

	return func(yield func(*finding) bool) {
		finding := &finding{}
		switch {
		case queryDeploymentData:
			_ = queryDeployments(ctx, ds.deployments, query, finding, yield)
		case queryImageData:
			_ = queryImages(ctx, ds.images, query, finding, yield)
		}
	}
}

func queryDeployments(ctx context.Context, ds deploymentDS.DataStore, query *v1.Query, finding *finding, yield func(*finding) bool) error {
	return ds.WalkByQuery(ctx, query, func(deployment *storage.Deployment) error {
		finding.deployment = deployment
		images, err := ds.GetImagesForDeployment(ctx, deployment)
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
}

func queryImages(ctx context.Context, ds imageDS.DataStore, query *v1.Query, finding *finding, yield func(*finding) bool) error {
	return ds.WalkByQuery(ctx, query, func(image *storage.Image) error {
		finding.image = image
		if !forEachFinding(yield, finding) {
			return common.ErrStopIterator
		}
		return nil
	})
}

func queryDeploymentData(mle common.MetricLabelsExpressions) bool {
	for _, expr := range mle {
		for label := range expr {
			if slices.Contains(deploymentLabels, label) {
				return true
			}
		}
	}
	return false
}

func queryImageData(mle common.MetricLabelsExpressions) bool {
	for _, expr := range mle {
		for label := range expr {
			if slices.Contains(imageLabels, label) {
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
