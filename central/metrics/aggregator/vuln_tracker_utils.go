package aggregator

import (
	"context"
	"strconv"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var labelOrder = map[Label]int{
	"Cluster":          1,
	"Namespace":        2,
	"Deployment":       3,
	"ImageID":          4,
	"ImageRegistry":    5,
	"ImageRemote":      6,
	"ImageTag":         7,
	"Component":        8,
	"ComponentVersion": 9,
	"CVE":              10,
	"CVSS":             11,
	"OperatingSystem":  12,
	"Severity":         13,
	"SeverityV2":       14,
	"SeverityV3":       15,
	"IsFixable":        16,
}

func trackVulnerabilityMetrics(ctx context.Context, ds deploymentDS.DataStore, mc metricsConfig) result {
	aggregated := make(result)
	for metric := range mc {
		aggregated[metric] = make(map[metricKey]*record)
	}
	// Optimization opportunity:
	// The resource filter is known at this point, so a more precise query could be constructed here.
	_ = ds.WalkByQuery(ctx, search.EmptyQuery(), func(deployment *storage.Deployment) error {
		images, err := ds.GetImagesForDeployment(ctx, deployment)
		if err != nil {
			return nil
		}
		return trackDeployment(mc, aggregated, deployment, images)
	})
	return aggregated
}

func trackDeployment(mc metricsConfig, aggregated result, deployment *storage.Deployment, images []*storage.Image) error {

	forEachVuln(images, func(image *storage.Image, imageName *storage.ImageName, component *storage.EmbeddedImageScanComponent, vuln *storage.EmbeddedVulnerability) {
		labelGetter := makeLabelGetter(image, imageName, component, vuln,
			deployment.GetClusterName(),
			deployment.GetNamespace(),
			deployment.GetName())

		for metric, expressions := range mc {
			if key, labels := makeAggregationKeyInstance(expressions, labelGetter); key != "" {
				if rec, ok := aggregated[metric][key]; ok {
					rec.total++
				} else {
					aggregated[metric][key] = &record{
						labels: labels,
						total:  1,
					}
				}
			}
		}
	})

	return nil
}

func forEachVuln(images []*storage.Image, f func(*storage.Image, *storage.ImageName, *storage.EmbeddedImageScanComponent, *storage.EmbeddedVulnerability)) {
	for _, image := range images {
		for _, component := range image.GetScan().GetComponents() {
			for _, vuln := range component.GetVulns() {
				for _, name := range image.GetNames() {
					f(image, name, component, vuln)
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

func makeLabelGetter(image *storage.Image, name *storage.ImageName, component *storage.EmbeddedImageScanComponent, vuln *storage.EmbeddedVulnerability, clusterName string, namespaceName string, deploymentName string) func(Label) string {
	return func(label Label) string {
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

func getResourceLabel(label Label, clusterName, namespaceName, deploymentName string) string {
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

func getImageComponentLabel(label Label, image *storage.Image, name *storage.ImageName, component *storage.EmbeddedImageScanComponent) string {
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

func getVulnerabilityLabel(label Label, image *storage.Image, vuln *storage.EmbeddedVulnerability) string {
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
