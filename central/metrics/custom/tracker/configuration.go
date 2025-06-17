package tracker

import (
	"slices"
	"time"
)

type Label string      // Prometheus label.
type MetricName string // Prometheus metric name.

// MetricsConfiguration is the parsed aggregation configuration.
type MetricsConfiguration map[MetricName][]Label

// diff computes the difference between one instance of MetricsConfiguration and
// another. The result serves for runtime updates.
func (mcfg MetricsConfiguration) diff(another MetricsConfiguration) (toAdd []MetricName, toDelete []MetricName, changed []MetricName) {
	for metricName, labels := range mcfg {
		if anotherLabels, ok := another[metricName]; !ok {
			toDelete = append(toDelete, metricName)
		} else if !slices.Equal(labels, anotherLabels) {
			changed = append(changed, metricName)
		}
	}
	for metricName := range another {
		if _, ok := mcfg[metricName]; !ok {
			toAdd = append(toAdd, metricName)
		}
	}
	return toAdd, toDelete, changed
}

type Configuration struct {
	metrics  MetricsConfiguration
	toAdd    []MetricName
	toDelete []MetricName
	period   time.Duration
}
