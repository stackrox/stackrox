package tracker

import (
	"slices"
	"time"
)

type Label string      // Prometheus label.
type MetricName string // Prometheus metric name.

// MetricDescriptors is the parsed aggregation configuration.
type MetricDescriptors map[MetricName][]Label

// diff computes the difference between one instance of MetricDescriptors and
// another. The result serves for runtime updates.
func (md MetricDescriptors) diff(another MetricDescriptors) (toAdd []MetricName, toDelete []MetricName, changed []MetricName) {
	for metricName, labels := range md {
		if anotherLabels, ok := another[metricName]; !ok {
			toDelete = append(toDelete, metricName)
		} else if !slices.Equal(labels, anotherLabels) {
			changed = append(changed, metricName)
		}
	}
	for metricName := range another {
		if _, ok := md[metricName]; !ok {
			toAdd = append(toAdd, metricName)
		}
	}
	return toAdd, toDelete, changed
}

type Configuration struct {
	metrics  MetricDescriptors
	toAdd    []MetricName
	toDelete []MetricName
	period   time.Duration
}
