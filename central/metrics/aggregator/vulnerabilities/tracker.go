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

var labelOrder = common.MakeLabelOrderMap([]common.Label{
	"Cluster",
	"Namespace",
	"Deployment",
	"ImageID",
	"ImageRegistry",
	"ImageRemote",
	"ImageTag",
	"Component",
	"ComponentVersion",
	"CVE",
	"CVSS",
	"OperatingSystem",
	"Severity",
	"SeverityV2",
	"SeverityV3",
	"IsFixable",
})

var getters = map[common.Label]func(*finding) string{
	"Cluster":          func(f *finding) string { return f.deployment.GetClusterName() },
	"Namespace":        func(f *finding) string { return f.deployment.GetNamespace() },
	"Deployment":       func(f *finding) string { return f.deployment.GetName() },
	"ImageID":          func(f *finding) string { return f.image.GetId() },
	"ImageRegistry":    func(f *finding) string { return f.name.GetRegistry() },
	"ImageRemote":      func(f *finding) string { return f.name.GetRemote() },
	"ImageTag":         func(f *finding) string { return f.name.GetTag() },
	"Component":        func(f *finding) string { return f.component.GetName() },
	"ComponentVersion": func(f *finding) string { return f.component.GetVersion() },
	"CVE":              func(f *finding) string { return f.vuln.GetCve() },
	"CVSS":             func(f *finding) string { return strconv.FormatFloat(float64(f.vuln.GetCvss()), 'f', 1, 32) },
	"OperatingSystem":  func(f *finding) string { return f.image.GetScan().GetOperatingSystem() },
	"Severity":         func(f *finding) string { return f.vuln.GetSeverity().String() },
	"SeverityV2":       func(f *finding) string { return f.vuln.GetCvssV2().GetSeverity().String() },
	"SeverityV3":       func(f *finding) string { return f.vuln.GetCvssV3().GetSeverity().String() },
	"IsFixable": func(f *finding) string {
		if f.vuln.GetFixedBy() == "" {
			return "false"
		}
		return "true"
	},
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
		labelOrder,
		getters,
		common.Bind3rd(trackVulnerabilityMetrics, deploymentDS.Singleton()),
		gauge)
	return tc
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
