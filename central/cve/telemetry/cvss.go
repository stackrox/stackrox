package telemetry

import (
	"context"
	"strconv"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

func (h *trackImpl) trackCvssMetrics(ctx context.Context) {
	aggregated := map[aggregationKey]map[keyInstance]int{}
	for key := range keysMap {
		aggregated[key] = make(map[keyInstance]int)
	}
	_ = h.ds.WalkByQuery(ctx, search.EmptyQuery(), func(deployment *storage.Deployment) error {
		return h.trackDeployment(ctx, aggregated, deployment)
	})
	h.aggregated(aggregated)
}

func (h *trackImpl) trackDeployment(ctx context.Context, aggregated map[string]map[string]int, deployment *storage.Deployment) error {
	images, err := h.ds.GetImagesForDeployment(ctx, deployment)
	if err != nil {
		return nil
	}

	forEachVuln(images, func(image *storage.Image, name *storage.ImageName, vuln *storage.EmbeddedVulnerability) {
		metric := makeCvssMetric(image, name, vuln,
			deployment.GetClusterName(), deployment.GetNamespace(), deployment.GetName())

		for key, expressions := range keysMap {
			if k := makeAggregationKeyInstance(expressions, metric); k != "" {
				aggregated[key][k]++
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

func makeCvssMetric(image *storage.Image, name *storage.ImageName, vuln *storage.EmbeddedVulnerability, clusterName string, namespaceName string, deploymentName string) map[string]string {
	return map[string]string{
		"Cluster":       clusterName,
		"Namespace":     namespaceName,
		"Deployment":    deploymentName,
		"ImageId":       image.GetId(),
		"ImageRegistry": name.GetRegistry(),
		"ImageRemote":   name.GetRemote(),
		"ImageTag":      name.GetTag(),

		"CVE":             vuln.GetCve(),
		"CVSS":            strconv.FormatFloat(float64(vuln.GetCvss()), 'f', 1, 32),
		"OperatingSystem": image.GetScan().GetOperatingSystem(),
		"Severity":        vuln.GetSeverity().String(),
		"SeverityV2":      vuln.GetCvssV2().GetSeverity().String(),
		"SeverityV3":      vuln.GetCvssV3().GetSeverity().String(),
		"IsFixable":       isFixable(vuln),
	}
}
