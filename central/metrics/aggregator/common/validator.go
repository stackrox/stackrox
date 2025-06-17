package common

import (
	"maps"
	"regexp"
	"slices"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

var (
	errInvalidConfiguration = errox.InvalidArgs.New("invalid configuration")

	// Source: https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
	metricNamePattern = regexp.MustCompile("^[a-zA-Z_:][a-zA-Z0-9_:]*$")
)

func isKnownLabel(label string, labelOrder map[Label]int) bool {
	_, ok := labelOrder[Label(label)]
	return ok
}

// validateMetricName ensures the name is alnum_.
func validateMetricName(name string) error {
	if len(name) == 0 {
		return errors.New("empty")
	}
	if !metricNamePattern.MatchString(name) {
		return errors.New(`doesn't match "` + metricNamePattern.String() + `"`)
	}
	return nil
}

// TranslateMetricLabels converts the storage object to the usable map,
// validating the values.
func TranslateMetricLabels(config map[string]*storage.PrometheusMetrics_MetricGroup_Labels, labelOrder map[Label]int) (MetricsConfiguration, error) {
	result := make(MetricsConfiguration, len(config))
	for metric, labels := range config {
		if err := validateMetricName(metric); err != nil {
			return nil, errInvalidConfiguration.CausedByf(
				"invalid metric name %q: %v", metric, err)
		}
		metricLabels := make([]Label, 0, len(labels.GetLabels()))
		for _, label := range labels.GetLabels() {
			if !isKnownLabel(label, labelOrder) {
				return nil, errInvalidConfiguration.CausedByf(
					"label %q for metric %q is not in the list of known labels: %v",
					label, metric, slices.Sorted(maps.Keys(labelOrder)))
			}
			metricLabels = append(metricLabels, Label(label))
		}
		if len(metricLabels) > 0 {
			result[MetricName(metric)] = metricLabels
		}
	}
	return result, nil
}
