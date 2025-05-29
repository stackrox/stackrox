package image_vulnerabilities

import (
	"context"
	"iter"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	imageCVEDS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/central/platform/matcher"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

var (
	getters = []common.LabelGetter[*finding]{
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

	deploymentLabels = []common.Label{"Cluster", "Namespace", "Deployment", "IsPlatformWorkload"}
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

func isPlatformWorkload(f *finding) string {
	p, _ := matcher.Singleton().MatchDeployment(f.deployment)
	return strconv.FormatBool(p)
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
	queryDeploymentData := mle.HasAnyLabelOf(deploymentLabels)
	queryImageData := !queryDeploymentData && mle.HasAnyLabelOf(imageLabels)

	return func(yield func(*finding) bool) {
		switch {
		case queryDeploymentData:
			_ = queryDeployments(ctx, ds.deployments, query, yield)
		case queryImageData:
			_ = queryImages(ctx, ds.images, query, yield)
		default:
			_ = queryCVEs(ctx, ds.cves, query, yield)
		}
	}
}

func queryDeployments(ctx context.Context, ds deploymentDS.DataStore, query *v1.Query, yield func(*finding) bool) error {
	finding := &finding{}
	return ds.WalkByQuery(ctx, query, func(deployment *storage.Deployment) error {
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

func queryImages(ctx context.Context, ds imageDS.DataStore, query *v1.Query, yield func(*finding) bool) error {
	finding := &finding{}
	return ds.WalkByQuery(ctx, query, func(image *storage.Image) error {
		finding.image = image
		if !forEachImageVuln(yield, finding) {
			return common.ErrStopIterator
		}
		return nil
	})
}

func convertCVE(ic *storage.ImageCVEV2) *storage.EmbeddedVulnerability {
	return &storage.EmbeddedVulnerability{
		Cve:      ic.GetCveBaseInfo().GetCve(),
		Severity: ic.GetSeverity(),
		Cvss:     ic.GetCvss(),
		CvssV2:   ic.GetCveBaseInfo().GetCvssV2(),
		CvssV3:   ic.GetCveBaseInfo().GetCvssV3(),
		SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: ic.GetFixedBy(),
		},
	}
}

func queryCVEs(ctx context.Context, ds imageCVEDS.DataStore, query *v1.Query, yield func(*finding) bool) error {
	finding := &finding{}
	return ds.WalkByQuery(ctx, query, func(ic *storage.ImageCVEV2) error {
		finding.vuln = convertCVE(ic)
		if !yield(finding) {
			return common.ErrStopIterator
		}
		return nil
	})
}

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
