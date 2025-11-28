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
	metrics  MetricDescriptors
	filters  LabelFilters
	toAdd    []MetricName
	toDelete []MetricName
	period   time.Duration
}

func (c *Configuration) GetMetricDescriptors() MetricDescriptors {
	if c != nil {
		return c.metrics
	}
	return nil
}

func (c *Configuration) GetLabelFilters() LabelFilters {
	if c != nil {
		return c.filters
	}
	return nil
}

// AllMetricsHaveFilter checks if all metrics in the configuration have a filter
// for the given label that matches the given pattern.
func (c *Configuration) AllMetricsHaveFilter(label Label, pattern string) bool {
	pattern = fullMatchPattern(pattern)
	if c == nil {
		return false
	}
	for metricName := range c.metrics {
		labelFilters, ok := c.filters[metricName]
		if !ok {
			return false
		}
		if expr, ok := labelFilters[label]; !ok || expr.String() != pattern {
			return false
		}
	}
	return len(c.metrics) != 0
}
