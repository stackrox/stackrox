package tracker

import (
	"regexp"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

var (
	errInvalidConfiguration = errox.InvalidArgs.New("invalid configuration")

	// Source: https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
	metricNamePattern = regexp.MustCompile("^[a-zA-Z_:][a-zA-Z0-9_:]*$")
)

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
func (tracker *TrackerBase[F]) validateLabels(labels []string, metricName string) ([]Label, error) {
	if len(labels) == 0 {
		return nil, errInvalidConfiguration.CausedByf("no labels specified for metric %q", metricName)
	}
	metricLabels := make([]Label, 0, len(labels))
	for _, label := range labels {
		validated, err := tracker.validateLabel(label, metricName)
		if err != nil {
			return nil, err
		}
		metricLabels = append(metricLabels, validated)
	}
	slices.Sort(metricLabels)
	return metricLabels, nil
}

func (tracker *TrackerBase[F]) validateLabel(label string, metricName string) (Label, error) {
	if _, ok := tracker.getters[Label(label)]; !ok {
		return "", errInvalidConfiguration.CausedByf("label %q for metric %q is not in the list of known labels %v", label,
			metricName, tracker.getters.GetLabels())
	}
	return Label(label), nil
}

// translateStorageConfiguration converts the storage object to the usable map,
// validating the values.
func (tracker *TrackerBase[F]) translateStorageConfiguration(config map[string]*storage.PrometheusMetrics_Group_Labels) (MetricDescriptors, LabelFilters, error) {
	result := make(MetricDescriptors, len(config))
	metricPrefix := tracker.metricPrefix
	if metricPrefix != "" {
		metricPrefix += "_"
	}
	filters := make(LabelFilters)
	for metricName, labels := range config {
		metricName = metricPrefix + metricName
		if err := validateMetricName(metricName); err != nil {
			return nil, nil, errInvalidConfiguration.CausedByf(
				"invalid metric name %q: %v", metricName, err)
		}
		metricLabels, err := tracker.validateLabels(labels.GetLabels(), metricName)
		if err != nil {
			return nil, nil, err
		}

		if len(labels.GetFilters()) > 0 {
			patterns := make(map[Label]*regexp.Regexp)
			for label, pattern := range labels.GetFilters() {
				validated, err := tracker.validateLabel(label, metricName)
				if err != nil {
					return nil, nil, err
				}
				if !strings.HasPrefix(pattern, "^") {
					pattern = "^" + pattern
				}
				if !strings.HasSuffix(pattern, "$") {
					pattern = pattern + "$"
				}
				patterns[validated], err = regexp.Compile(pattern)
				if err != nil {
					return nil, nil, errInvalidConfiguration.CausedByf(
						"bad filter expression for metric %q label %q: %v",
						metricName, label, err)
				}
			}
			filters[MetricName(metricName)] = patterns
		}
		result[MetricName(metricName)] = metricLabels
	}
	return result, filters, nil
}
