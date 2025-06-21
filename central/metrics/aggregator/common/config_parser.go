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

// ParseMetricLabels converts the storage object to the usable map, validating
// the values.
func ParseMetricLabels(config map[string]*storage.PrometheusMetricsConfig_Labels, labelOrder map[Label]int) (MetricsConfiguration, error) {
	result := make(MetricsConfiguration, len(config))
	for metric, labels := range config {
		if err := validateMetricName(metric); err != nil {
			return nil, errInvalidConfiguration.CausedByf(
				"invalid metric name %q: %v", metric, err)
		}
		labelExpression := make(map[Label]Expression, len(labels.GetLabels()))
		for label, expression := range labels.GetLabels() {
			if !isKnownLabel(label, labelOrder) {
				return nil, errInvalidConfiguration.CausedByf(
					"label %q for metric %q is not in the list of known labels: %v",
					label, metric, slices.Sorted(maps.Keys(labelOrder)))
			}

			var expr Expression
			for _, cond := range expression.GetExpression() {
				condition, err := MakeCondition(cond.GetOperator(), cond.GetArgument())
				if err != nil {
					return nil, errInvalidConfiguration.CausedByf(
						"failed to parse a condition for metric %q with label %q: %v",
						metric, label, err)
				}
				expr = append(expr, condition)
			}
			labelExpression[Label(label)] = expr
		}
		if len(labelExpression) > 0 {
			result[MetricName(metric)] = labelExpression
		}
	}
	return result, nil
}
