package telemetry

import (
	"errors"
	"regexp"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

var metricNamePattern = regexp.MustCompile("^[a-zA-Z0-9_]+$")

func validateMetricName(s string) error {
	if len(s) == 0 {
		return errors.New("empty")
	}
	if !metricNamePattern.MatchString(s) {
		return errors.New("bad characters")
	}
	return nil
}

// parseConfig converts the storage object to the usable map, validating the values.
func parseConfig(config *storage.PrometheusMetricsConfig) (metricsConfig, time.Duration, error) {

	period := time.Hour * time.Duration(config.GetGatheringPeriodHours())
	if period == 0 {
		return nil, period, nil
	}
	result := make(metricsConfig)
	for metric, labels := range config.GetMetricLabels() {
		if err := validateMetricName(metric); err != nil {
			return nil, 0, errox.InvalidArgs.CausedByf(
				"invalid metric name %q: %v", metric, err)
		}
		metricLabels := make(map[Label][]*expression)
		for label, expressions := range labels.GetLabelExpressions() {

			if _, knownLabel := labelOrder[Label(label)]; !knownLabel {
				return nil, 0, errox.InvalidArgs.CausedByf("unknown label %q for metric %q", label, metric)
			}

			var exprs []*expression
			for _, expr := range expressions.GetExpression() {
				expr := &expression{
					op:  operator(expr.GetOperator()),
					arg: expr.GetArgument(),
				}
				if err := expr.validate(); err != nil {
					return nil, 0, errox.InvalidArgs.CausedByf(
						"failed to parse expression for metric %q with label %q: %v",
						metric, label, err)
				}
				exprs = append(exprs, expr)
			}
			metricLabels[Label(label)] = exprs
		}
		result[metricName(metric)] = metricLabels
	}
	if len(result) == 0 {
		return nil, 0, nil
	}
	return result, period, nil
}
