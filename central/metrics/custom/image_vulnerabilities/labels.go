package image_vulnerabilities

import (
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
)

var (
	lazyLabels = []tracker.LazyLabel[finding]{
		{Label: "Cluster", Getter: func(f *finding) string { return f.deployment.GetClusterName() }},
	}

	labels = tracker.MakeLabelOrderMap(lazyLabels)
)

type finding struct {
	deployment *storage.Deployment
}

func ValidateConfiguration(config map[string]*storage.PrometheusMetrics_MetricGroup_Labels) error {
	_, err := tracker.TranslateConfiguration(config, labels)
	return err
}
