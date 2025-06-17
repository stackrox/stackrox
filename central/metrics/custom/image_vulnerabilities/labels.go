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

// finding holds all information for computing any label in this category.
// The aggregator calls the lazy label's Getter function with every finding to
// compute the values for the list of defined labels.
type finding struct {
	deployment *storage.Deployment
}

func ValidateConfiguration(config map[string]*storage.PrometheusMetrics_Group_Labels) error {
	_, err := tracker.TranslateConfiguration(config, labels)
	return err
}
