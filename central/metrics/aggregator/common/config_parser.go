package common

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

func isKnownLabel(label string, labelOrder map[Label]int) bool {
	_, ok := labelOrder[Label(label)]
	return ok
}

// parseMetricLabels converts the storage object to the usable map, validating the values.
func parseMetricLabels(config map[string]*storage.PrometheusMetricsConfig_MetricLabels, labelOrder map[Label]int) (MetricLabelsExpressions, error) {

	result := make(MetricLabelsExpressions)
	for metric, labels := range config {
		if err := validateMetricName(metric); err != nil {
			return nil, errox.InvalidArgs.CausedByf(
				"invalid metric name %q: %v", metric, err)
		}
		metricLabels := make(map[Label][]*Expression)
		for label, expressions := range labels.GetLabelExpressions() {

			if !isKnownLabel(label, labelOrder) {
				return nil, errox.InvalidArgs.CausedByf("unknown label %q for metric %q", label, metric)
			}

			var exprs []*Expression
			for _, expr := range expressions.GetExpression() {
				if expr, err := MakeExpression(expr.GetOperator(), expr.GetArgument()); err != nil {
					return nil, errox.InvalidArgs.CausedByf(
						"failed to parse expression for metric %q with label %q: %v",
						metric, label, err)
				} else {
					exprs = append(exprs, expr)
				}
			}
			metricLabels[Label(label)] = exprs
		}
		result[MetricName(metric)] = metricLabels
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}
