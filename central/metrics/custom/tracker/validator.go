package tracker

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

// validateLabels checks if the labels exist in the labelOrder map and returns
// a sorted label list.
func validateLabels(labels []string, labelOrder map[Label]int, metricName string) ([]Label, error) {
	if len(labels) == 0 {
		return nil, errInvalidConfiguration.CausedByf("no labels specified for metric %q", metricName)
	}
	metricLabels := make([]Label, 0, len(labels))
	for _, label := range labels {
		if !isKnownLabel(label, labelOrder) {
			return nil, errInvalidConfiguration.CausedByf("label %q for metric %q is not in the list of known labels %v", label,
				metricName, slices.Sorted(maps.Keys(labelOrder)))
		}
		metricLabels = append(metricLabels, Label(label))
	}
	slices.SortFunc(metricLabels, func(a, b Label) int {
		return labelOrder[Label(a)] - labelOrder[Label(b)]
	})
	return metricLabels, nil
}

// translateStorageConfiguration converts the storage object to the usable map,
// validating the values.
func translateStorageConfiguration(config map[string]*storage.PrometheusMetrics_Group_Labels, metricPrefix string, labelOrder map[Label]int) (MetricDescriptors, error) {
	result := make(MetricDescriptors, len(config))
	if metricPrefix != "" {
		metricPrefix += "_"
	}
	for metricName, labels := range config {
		metricName = metricPrefix + metricName
		if err := validateMetricName(metricName); err != nil {
			return nil, errInvalidConfiguration.CausedByf(
				"invalid metric name %q: %v", metricName, err)
		}
		metricLabels, err := validateLabels(labels.GetLabels(), labelOrder, metricName)
		if err != nil {
			return nil, err
		}
		result[MetricName(metricName)] = metricLabels
	}
	return result, nil
}
