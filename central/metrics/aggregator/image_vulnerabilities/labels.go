package image_vulnerabilities

import (
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/generated/storage"
)

var (
	lazyLabels = []common.LazyLabel[finding]{
		{Label: "Cluster", Getter: func(f *finding) string { return f.deployment.GetClusterName() }},
	}

	labels = common.MakeLabelOrderMap(lazyLabels)
)

type finding struct {
	deployment *storage.Deployment
}

func ValidateConfiguration(config map[string]*storage.PrometheusMetrics_MetricGroup_Labels) error {
	_, err := common.TranslateMetricLabels(config, labels)
	return err
}
