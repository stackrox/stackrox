package aggregator

import (
	"github.com/stackrox/rox/central/metrics/custom/image_vulnerabilities"
	"github.com/stackrox/rox/generated/storage"
)

func ValidateConfiguration(config *storage.PrometheusMetrics) error {
	return image_vulnerabilities.ValidateConfiguration(
		config.GetImageVulnerabilities().GetDescriptors())
}
