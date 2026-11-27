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

// validateLabels checks if the labels exist in the known labels and returns
// a sorted label list.
func validateLabels(labels []string, metricName string, knownLabels []Label) ([]Label, error) {
	if len(labels) == 0 {
		return nil, errInvalidConfiguration.CausedByf("no labels specified for metric %q", metricName)
	}
	metricLabels := make([]Label, 0, len(labels))
	for _, label := range labels {
		validated, err := validateLabel(label, metricName, knownLabels)
		if err != nil {
			return nil, err
		}
		metricLabels = append(metricLabels, validated)
	}
	slices.Sort(metricLabels)
	return metricLabels, nil
}

func validateLabel(label string, metricName string, knownLabels []Label) (Label, error) {
	if slices.Contains(knownLabels, Label(label)) {
		return Label(label), nil
	}
	return "", errInvalidConfiguration.CausedByf("label %q for metric %q is not in the list of known labels %v", label,
		metricName, knownLabels)
}

// parseFilters parses a map of label names to regex patterns, validating each label and pattern.
func parseFilters(filters map[string]string, metricName, filterType string, knownLabels []Label) (map[Label]*regexp.Regexp, error) {
	if len(filters) == 0 {
		return nil, nil
	}
	patterns := make(map[Label]*regexp.Regexp, len(filters))
	for label, pattern := range filters {
		validated, err := validateLabel(label, metricName, knownLabels)
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(pattern, "^") {
			pattern = "^" + pattern
		}
		if !strings.HasSuffix(pattern, "$") {
			pattern = pattern + "$"
		}
		patterns[validated], err = regexp.Compile(pattern)
		if err != nil {
			return nil, errInvalidConfiguration.CausedByf(
				"bad %s expression for metric %q label %q: %v",
				filterType, metricName, label, err)
		}
	}
	return patterns, nil
}

// translateStorageConfiguration converts the storage object to the usable map,
// validating the values.
func translateStorageConfiguration(config map[string]*storage.PrometheusMetrics_Group_Labels, metricPrefix string, knownLabels []Label) (MetricDescriptors, LabelFilters, LabelFilters, error) {
	result := make(MetricDescriptors, len(config))
	if metricPrefix != "" {
		metricPrefix += "_"
	}
	includeFilters := make(LabelFilters)
	excludeFilters := make(LabelFilters)
	for descriptorKey, labels := range config {
		metricName := metricPrefix + descriptorKey
		if err := validateMetricName(metricName); err != nil {
			return nil, nil, nil, errInvalidConfiguration.CausedByf(
				"invalid metric name %q: %v", metricName, err)
		}
		metricLabels, err := validateLabels(labels.GetLabels(), metricName, knownLabels)
		if err != nil {
			return nil, nil, nil, err
		}

		incPatterns, err := parseFilters(labels.GetIncludeFilters(), metricName, "include_filter", knownLabels)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(incPatterns) > 0 {
			includeFilters[MetricName(metricName)] = incPatterns
		}

		excPatterns, err := parseFilters(labels.GetExcludeFilters(), metricName, "exclude_filter", knownLabels)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(excPatterns) > 0 {
			excludeFilters[MetricName(metricName)] = excPatterns
		}

		result[MetricName(metricName)] = metricLabels
	}
	return result, includeFilters, excludeFilters, nil
}
