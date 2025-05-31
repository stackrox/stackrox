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
func parseMetricLabels(config map[string]*storage.PrometheusMetricsConfig_MetricLabels, labelOrder map[Label]int) (MetricLabelsExpressions, error) {

	result := make(MetricLabelsExpressions)
	for metric, labels := range config {
		if err := validateMetricName(metric); err != nil {
			return nil, errInvalidConfiguration.CausedByf(
				"invalid metric name %q: %v", metric, err)
		}
		metricLabels := make(map[Label][]*Expression)
		for label, expressions := range labels.GetLabelExpressions() {

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

			var exprs []*Expression
			for _, expr := range expressions.GetExpression() {
				if expr, err := MakeExpression(expr.GetOperator(), expr.GetArgument()); err != nil {
					return nil, errInvalidConfiguration.CausedByf(
						"failed to parse expression for metric %q with label %q: %v",
						metric, label, err)
				} else {
					exprs = append(exprs, expr)
				}
			}
			metricLabels[Label(label)] = exprs
		}
		if len(metricLabels) > 0 {
			result[MetricName(metric)] = metricLabels
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}
