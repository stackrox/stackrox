package aggregator

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
)

const vulnerabilitiesCategory = "vulnerabilities"

func reloadVulnerabilityTrackerConfig(cfg *storage.PrometheusMetricsConfig_Vulnerabilities) (*tracker, error) {
	vulnTracker := makeTracker(vulnerabilitiesCategory)
	metricsConfig, period, err := parseVulnerabilitiesConfig(cfg)
	if err != nil {
		log.Errorw("Failed to parse vulnerability metrics configuration", logging.Err(err))
		return vulnTracker, err
	}
	if period == 0 {
		log.Info("Vulnerability metrics collection is disabled")
	}
	vulnTracker.reloadConfig(metricsConfig, period)
	return vulnTracker, nil
}

// parseVulnerabilitiesConfig converts the storage object to the usable map, validating the values.
func parseVulnerabilitiesConfig(config *storage.PrometheusMetricsConfig_Vulnerabilities) (metricsConfig, time.Duration, error) {

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
