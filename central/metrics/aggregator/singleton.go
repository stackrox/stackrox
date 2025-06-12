package aggregator

import (
	"github.com/stackrox/rox/central/metrics/aggregator/image_vulnerabilities"
	"github.com/stackrox/rox/generated/storage"
)

func ParseConfiguration(config *storage.PrometheusMetricsConfig) error {
	return image_vulnerabilities.ParseConfiguration(
		config.GetImageVulnerabilities())
}
