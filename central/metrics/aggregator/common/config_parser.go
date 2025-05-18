package common

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

// parseMetricLabels converts the storage object to the usable map, validating the values.
func parseMetricLabels(config map[string]*storage.PrometheusMetricsConfig_LabelExpressions, labelOrder map[Label]int) (MetricLabelExpressions, error) {

	result := make(MetricLabelExpressions)
	for metric, labels := range config {
		if err := validateMetricName(metric); err != nil {
			return nil, errox.InvalidArgs.CausedByf(
				"invalid metric name %q: %v", metric, err)
		}
		metricLabels := make(map[Label][]*Expression)
		for label, expressions := range labels.GetLabelExpressions() {

			if _, knownLabel := labelOrder[Label(label)]; !knownLabel {
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
