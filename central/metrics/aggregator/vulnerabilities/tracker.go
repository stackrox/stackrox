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
	return common.MakeTrackerConfig("vulnerabilities", "aggregated CVEs",
		labelOrder, common.Bind2nd(trackVulnerabilityMetrics, deploymentDS.Singleton()))
}

func trackVulnerabilityMetrics(ctx context.Context, ds deploymentDS.DataStore) iter.Seq[common.Finding] {

	return func(yield func(common.Finding) bool) {
		// Optimization opportunity:
		// The resource filter is known at this point, so a more precise query could be constructed here.
		_ = ds.WalkByQuery(ctx, search.EmptyQuery(), func(deployment *storage.Deployment) error {
			images, err := ds.GetImagesForDeployment(ctx, deployment)
			if err != nil {
				return nil
			}
			for finding := range vulnerabitilies(images, deployment) {
				if !yield(finding) {
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
							deployment.GetClusterName(),
							deployment.GetNamespace(),
							deployment.GetName(),
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

func isFixable(vuln *storage.EmbeddedVulnerability) string {
	if vuln.GetFixedBy() == "" {
		return "false"
	}
	return "true"
}

func makeFinding(
	clusterName string,
	namespaceName string,
	deploymentName string,
	image *storage.Image,
	name *storage.ImageName,
	component *storage.EmbeddedImageScanComponent,
	vuln *storage.EmbeddedVulnerability,
) common.Finding {

	return func(label common.Label) string {
		switch label {
		case "Cluster", "Namespace", "Deployment":
			return getResourceLabel(label, clusterName, namespaceName, deploymentName)
		case "ImageID", "ImageRegistry", "ImageRemote", "ImageTag", "Component", "ComponentVersion":
			return getImageComponentLabel(label, image, name, component)
		case "CVE", "CVSS", "OperatingSystem", "Severity", "SeverityV2", "SeverityV3", "IsFixable":
			return getVulnerabilityLabel(label, image, vuln)
		default:
			return ""
		}
	}
}

func getResourceLabel(label common.Label, clusterName, namespaceName, deploymentName string) string {
	switch label {
	case "Cluster":
		return clusterName
	case "Namespace":
		return namespaceName
	case "Deployment":
		return deploymentName
	default:
		return ""
	}
}

func getImageComponentLabel(label common.Label, image *storage.Image, name *storage.ImageName, component *storage.EmbeddedImageScanComponent) string {
	switch label {
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
	default:
		return ""
	}
}

func getVulnerabilityLabel(label common.Label, image *storage.Image, vuln *storage.EmbeddedVulnerability) string {
	switch label {
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
		return isFixable(vuln)
	default:
		return ""
	}
}
