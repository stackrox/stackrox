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

func (mcfg MetricsConfiguration) diffLabels(another MetricsConfiguration) (toAdd []MetricName, toDelete []MetricName, changed []MetricName) {
	for metric, labels := range mcfg {
		if anotherLabels, ok := another[metric]; !ok {
			toDelete = append(toDelete, metric)
		} else if !slices.Equal(labels, anotherLabels) {
			changed = append(changed, metric)
		}
	}
	for metric := range another {
		if _, ok := mcfg[metric]; !ok {
			toAdd = append(toAdd, metric)
		}
	}
	return toAdd, toDelete, changed
}
