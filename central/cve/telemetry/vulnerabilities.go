package telemetry

import (
	"context"
	"strconv"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Label is an alias because the central/metrics package cannot import it.
type Label = string

const (
	ClusterLabel          Label = "Cluster"
	NamespaceLabel        Label = "Namespace"
	DeploymentLabel       Label = "Deployment"
	ImageIDLabel          Label = "ImageID"
	ImageRegistryLabel    Label = "ImageRegistry"
	ImageRemoteLabel      Label = "ImageRemote"
	ImageTagLabel         Label = "ImageTag"
	ComponentLabel        Label = "Component"
	ComponentVersionLabel Label = "ComponentVersion"
	CVELabel              Label = "CVE"
	CVSSLabel             Label = "CVSS"
	OperatingSystemLabel  Label = "OperatingSystem"
	SeverityLabel         Label = "Severity"
	SeverityV2Label       Label = "SeverityV2"
	SeverityV3Label       Label = "SeverityV3"
	IsFixableLabel        Label = "IsFixable"
)

type record struct {
	labels map[Label]string
	total  int
}

type result map[metricName]map[metricKey]*record

func (h *vulnerabilityMetricsImpl) trackVulnerabilityMetrics(ctx context.Context) result {
	metrics := make(result)
	for metric := range h.metricExpressions {
		metrics[metric] = make(map[metricKey]*record)
	}
	// Optimization opportunity:
	// The resource filter is known at this point, so a more precise query could be constructed here.
	_ = h.ds.WalkByQuery(ctx, search.EmptyQuery(), func(deployment *storage.Deployment) error {
		return h.trackDeployment(ctx, metrics, deployment)
	})
	return metrics
}

func (h *vulnerabilityMetricsImpl) trackDeployment(ctx context.Context, aggregated result, deployment *storage.Deployment) error {
	images, err := h.ds.GetImagesForDeployment(ctx, deployment)
	if err != nil {
		return nil
	}

	forEachVuln(images, func(image *storage.Image, imageName *storage.ImageName, component *storage.EmbeddedImageScanComponent, vuln *storage.EmbeddedVulnerability) {
		labelGetter := makeLabelGetter(image, imageName, component, vuln,
			deployment.GetClusterName(),
			deployment.GetNamespace(),
			deployment.GetName())

		for metric, expressions := range h.metricExpressions {
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
		case ClusterLabel, NamespaceLabel, DeploymentLabel:
			return getResourceLabel(label, clusterName, namespaceName, deploymentName)
		case ImageIDLabel, ImageRegistryLabel, ImageRemoteLabel, ImageTagLabel, ComponentLabel, ComponentVersionLabel:
			return getImageComponentLabel(label, image, name, component)
		case CVELabel, CVSSLabel, OperatingSystemLabel, SeverityLabel, SeverityV2Label, SeverityV3Label, IsFixableLabel:
			return getVulnerabilityLabel(label, image, vuln)
		default:
			return ""
		}
	}
}

func getResourceLabel(label Label, clusterName, namespaceName, deploymentName string) string {
	switch label {
	case ClusterLabel:
		return clusterName
	case NamespaceLabel:
		return namespaceName
	case DeploymentLabel:
		return deploymentName
	default:
		return ""
	}
}

func getImageComponentLabel(label Label, image *storage.Image, name *storage.ImageName, component *storage.EmbeddedImageScanComponent) string {
	switch label {
	case ImageIDLabel:
		return image.GetId()
	case ImageRegistryLabel:
		return name.GetRegistry()
	case ImageRemoteLabel:
		return name.GetRemote()
	case ImageTagLabel:
		return name.GetTag()
	case ComponentLabel:
		return component.GetName()
	case ComponentVersionLabel:
		return component.GetVersion()
	default:
		return ""
	}
}

func getVulnerabilityLabel(label Label, image *storage.Image, vuln *storage.EmbeddedVulnerability) string {
	switch label {
	case CVELabel:
		return vuln.GetCve()
	case CVSSLabel:
		return strconv.FormatFloat(float64(vuln.GetCvss()), 'f', 1, 32)
	case OperatingSystemLabel:
		return image.GetScan().GetOperatingSystem()
	case SeverityLabel:
		return vuln.GetSeverity().String()
	case SeverityV2Label:
		return vuln.GetCvssV2().GetSeverity().String()
	case SeverityV3Label:
		return vuln.GetCvssV3().GetSeverity().String()
	case IsFixableLabel:
		return isFixable(vuln)
	default:
		return ""
	}
}
