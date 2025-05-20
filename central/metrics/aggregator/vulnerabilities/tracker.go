package vulnerabilities

import (
	"context"
	"iter"
	"strconv"

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

func MakeTrackerConfig() *common.TrackerConfig {
	tc := common.MakeTrackerConfig("vulnerabilities", "aggregated CVEs",
		labelOrder, common.Bind3rd(trackVulnerabilityMetrics, deploymentDS.Singleton()))
	return tc
}

func trackVulnerabilityMetrics(ctx context.Context, mle common.MetricLabelsExpressions, ds deploymentDS.DataStore) iter.Seq[common.Finding] {
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
	return func(yield func(common.Finding) bool) {
		// Optimization opportunity:
		// The resource filter is known at this point, so a more precise query could be constructed here.
		_ = ds.WalkByQuery(ctx, search.EmptyQuery(), func(deployment *storage.Deployment) error {
			if trackImageData {
				images, err := ds.GetImagesForDeployment(ctx, deployment)
				if err != nil {
					return nil
				}
				for finding := range vulnerabitilies(images, deployment) {
					if !yield(finding) {
						return common.ErrStopIterator
					}
				}
			} else {
				if !yield(func(label common.Label) string { return getResourceLabel(label, deployment) }) {
					return common.ErrStopIterator
				}
			}
			return nil
		})
	}
}

func vulnerabitilies(images []*storage.Image, deployment *storage.Deployment) iter.Seq[common.Finding] {
	return func(yield func(common.Finding) bool) {
		for _, image := range images {
			for _, component := range image.GetScan().GetComponents() {
				for _, vuln := range component.GetVulns() {
					for _, name := range image.GetNames() {
						finding := makeFinding(
							deployment,
							image,
							name,
							component,
							vuln,
						)
						if !yield(finding) {
							return
						}
					}
				}
			}
		}
	}
}

func makeFinding(
	deployment *storage.Deployment,
	image *storage.Image,
	name *storage.ImageName,
	component *storage.EmbeddedImageScanComponent,
	vuln *storage.EmbeddedVulnerability,
) common.Finding {

	return func(label common.Label) string {
		switch label {
		case "Cluster":
			return deployment.GetClusterName()
		case "Namespace":
			return deployment.GetNamespace()
		case "Deployment":
			return deployment.GetName()

		case "ImageID":
			return image.GetId()
		case "ImageRegistry":
			return name.GetRegistry()
		case "ImageRemote":
			return name.GetRemote()
		case "ImageTag":
			return name.GetTag()
		case "Component":
			return component.GetName()
		case "ComponentVersion":
			return component.GetVersion()

		case "CVE":
			return vuln.GetCve()
		case "CVSS":
			return strconv.FormatFloat(float64(vuln.GetCvss()), 'f', 1, 32)
		case "OperatingSystem":
			return image.GetScan().GetOperatingSystem()
		case "Severity":
			return vuln.GetSeverity().String()
		case "SeverityV2":
			return vuln.GetCvssV2().GetSeverity().String()
		case "SeverityV3":
			return vuln.GetCvssV3().GetSeverity().String()
		case "IsFixable":
			if vuln.GetFixedBy() == "" {
				return "false"
			}
			return "true"

		default:
			return ""
		}
	}
}

func getResourceLabel(label common.Label, deployment *storage.Deployment) string {
	switch label {
	case "Cluster":
		return deployment.GetClusterName()
	case "Namespace":
		return deployment.GetNamespace()
	case "Deployment":
		return deployment.GetName()
	default:
		return ""
	}
}
