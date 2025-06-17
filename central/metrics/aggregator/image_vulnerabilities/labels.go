package image_vulnerabilities

import (
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/generated/storage"
)

var (
	getters = []common.LabelGetter[*finding]{
		{Label: "Cluster", Getter: func(f *finding) string { return f.deployment.GetClusterName() }},
	}

	labels = common.MakeLabelOrderMap(getters)
)

type finding struct {
	common.OneOrMore
	deployment *storage.Deployment
}

func ParseConfiguration(config map[string]*storage.PrometheusMetrics_MetricGroup_Labels) error {
	_, err := common.TranslateMetricLabels(config, labels)
	return err
}
