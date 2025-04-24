package telemetry

import (
	"context"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/search"
)

var keysMap map[aggregationKey][]string

func parseAggregationKeys(setting string) map[aggregationKey][]string {
	result := make(map[aggregationKey][]string)
	for _, key := range strings.Split(setting, "|") {
		result[key] = strings.Split(key, ",")
	}
	return result
}

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

func makeAggregationKeyInstance(keys []aggregationKey, metric map[keyInstance]string) string {
	sb := strings.Builder{}
	for i, key := range keys {
		if v := metric[key]; v != "" {
			if i > 0 {
				sb.WriteRune('|')
			}
			sb.WriteString(v)
		}
	}
	return sb.String()
}

func (h *trackImpl) trackDeployment(ctx context.Context, aggregated map[string]map[string]int, deployment *storage.Deployment) error {
	images, err := h.ds.GetImagesForDeployment(ctx, deployment)
	if err != nil {
		return nil
	}

	reportCVSS := env.EnableCVSSMetrics.BooleanSetting()
	forEachVuln(images, func(image *storage.Image, name *storage.ImageName, vuln *storage.EmbeddedVulnerability) {
		metric := makeCvssMetric(image, name, vuln,
			deployment.GetClusterName(), deployment.GetNamespace())

		for key, keys := range keysMap {
			k := makeAggregationKeyInstance(keys, metric)
			aggregated[key][k]++
		}
		if reportCVSS {
			h.cvssGauge(metric, float64(vuln.GetCvss()))
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

func makeCvssMetric(image *storage.Image, name *storage.ImageName, vuln *storage.EmbeddedVulnerability, clusterName string, namespaceName string) map[string]string {
	return map[string]string{
		"Cluster":         clusterName,
		"Namespace":       namespaceName,
		"ImageId":         image.GetId(),
		"ImageRegistry":   name.GetRegistry(),
		"ImageRemote":     name.GetRemote(),
		"ImageTag":        name.GetTag(),
		"OperatingSystem": image.GetScan().GetOperatingSystem(),
		"CVE":             vuln.GetCve(),
		"Severity":        vuln.GetSeverity().String(),
		"SeverityV2":      vuln.GetCvssV2().GetSeverity().String(),
		"SeverityV3":      vuln.GetCvssV3().GetSeverity().String(),
		"IsFixable":       isFixable(vuln),
		// "DeploymentsCount": "0",
	}
}
