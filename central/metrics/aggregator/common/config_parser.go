package common

import (
	"slices"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

var errInvalidConfiguration = errox.InvalidArgs.New("invalid configuration")

func isKnownLabel(label string, labelOrder map[Label]int) bool {
	_, ok := labelOrder[Label(label)]
	return ok
}

// parseMetricLabels converts the storage object to the usable map, validating the values.
func parseMetricLabels(config map[string]*storage.PrometheusMetricsConfig_Labels, labelOrder map[Label]int) (MetricsConfiguration, error) {

	result := make(MetricsConfiguration)
	for metric, labels := range config {
		if err := validateMetricName(metric); err != nil {
			return nil, errInvalidConfiguration.CausedByf(
				"invalid metric name %q: %v", metric, err)
		}
		labelExpression := make(map[Label]Expression)
		for label, expression := range labels.GetLabels() {

			if !isKnownLabel(label, labelOrder) {
				var knownLabels []Label
				for k := range labelOrder {
					knownLabels = append(knownLabels, k)
				}
				slices.SortFunc(knownLabels, func(a, b Label) int {
					return labelOrder[a] - labelOrder[b]
				})
				return nil, errInvalidConfiguration.CausedByf(
					"label %q for metric %q is not in the list of known labels: %v", label, metric, knownLabels)
			}

			var expr Expression
			for _, condition := range expression.GetExpression() {
				if condition, err := MakeCondition(condition.GetOperator(), condition.GetArgument()); err != nil {
					return nil, errInvalidConfiguration.CausedByf(
						"failed to parse a condition for metric %q with label %q: %v",
						metric, label, err)
				} else {
					expr = append(expr, condition)
				}
			}
			labelExpression[Label(label)] = expr
		}
		if len(labelExpression) > 0 {
			result[MetricName(metric)] = labelExpression
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}
