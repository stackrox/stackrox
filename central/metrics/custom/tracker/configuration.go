package tracker

import (
	"regexp"
	"slices"
	"time"
)

type Label string      // Prometheus label.
type MetricName string // Prometheus metric name.

// MetricDescriptors is the parsed aggregation configuration.
type MetricDescriptors map[MetricName][]Label

// LabelFilters is a map of regex filters of the metric label values.
type LabelFilters map[MetricName]map[Label]*regexp.Regexp

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
	metrics        MetricDescriptors
	includeFilters LabelFilters
	excludeFilters LabelFilters
	toAdd          []MetricName
	toDelete       []MetricName
	period         time.Duration
}

func (cfg *Configuration) isEnabled() bool {
	if cfg == nil || len(cfg.metrics) == 0 {
		return false
	}
	return cfg.period > 0
}
