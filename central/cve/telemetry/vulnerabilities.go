package telemetry

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type record struct {
	labels map[string]string
	total  int
}

func (r *record) String() string {
	values := strings.Join(slices.Collect(maps.Values(r.labels)), ", ")
	return values + ": " + strconv.FormatInt(int64(r.total), 10)
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

	forEachVuln(images, func(image *storage.Image, imageName *storage.ImageName, vuln *storage.EmbeddedVulnerability) {
		labelGetter := makeVulnerabilityLabels(image, imageName, vuln,
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

func forEachVuln(images []*storage.Image, f func(*storage.Image, *storage.ImageName, *storage.EmbeddedVulnerability)) {
	for _, image := range images {
		for _, component := range image.GetScan().GetComponents() {
			for _, vuln := range component.GetVulns() {
				for _, name := range image.GetNames() {
					f(image, name, vuln)
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

func makeVulnerabilityLabels(image *storage.Image, name *storage.ImageName, vuln *storage.EmbeddedVulnerability, clusterName string, namespaceName string, deploymentName string) func(string) string {
	return func(label string) string {
		switch label {
		// Unique resource key (no component):
		case "Cluster":
			return clusterName
		case "Namespace":
			return namespaceName
		case "Deployment":
			return deploymentName
		case "ImageId":
			return image.GetId()
		case "ImageRegistry":
			return name.GetRegistry()
		case "ImageRemote":
			return name.GetRemote()
		case "ImageTag":
			return name.GetTag()
		// Values:
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
}
