package common

import (
	"slices"
)

type Label string      // Prometheus label.
type MetricName string // Prometheus metric name.

// MetricsConfiguration is the parsed aggregation configuration.
type MetricsConfiguration map[MetricName][]Label

func (mcfg MetricsConfiguration) hasAnyLabelOf(labels []Label) bool {
	for _, configLabels := range mcfg {
		for _, label := range configLabels {
			if slices.Contains(labels, label) {
				return true
			}
		}
	}
	return false
}
