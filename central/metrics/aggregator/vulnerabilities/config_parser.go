package vulnerabilities

import (
	"time"

	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
)

const vulnerabilitiesCategory = "vulnerabilities"

var log = logging.LoggerForModule()

func Reconfigure(cfg *storage.PrometheusMetricsConfig_Vulnerabilities) (*common.Tracker, error) {
	vulnTracker := common.MakeTracker(vulnerabilitiesCategory, "aggregated CVEs", labelOrder)
	metricsConfig, period, err := parseVulnerabilitiesConfig(cfg)
	if err != nil {
		log.Errorw("Failed to parse vulnerability metrics configuration", logging.Err(err))
		return vulnTracker, err
	}
	if period == 0 {
		log.Info("Vulnerability metrics collection is disabled")
	}
	vulnTracker.Reconfigure(metricsConfig, period)
	return vulnTracker, nil
}

// parseVulnerabilitiesConfig converts the storage object to the usable map, validating the values.
func parseVulnerabilitiesConfig(config *storage.PrometheusMetricsConfig_Vulnerabilities) (common.MetricsConfig, time.Duration, error) {

	period := time.Hour * time.Duration(config.GetGatheringPeriodHours())
	if period == 0 {
		return nil, period, nil
	}
	result := make(common.MetricsConfig)
	for metric, labels := range config.GetMetricLabels() {
		if err := common.ValidateMetricName(metric); err != nil {
			return nil, 0, errox.InvalidArgs.CausedByf(
				"invalid metric name %q: %v", metric, err)
		}
		metricLabels := make(map[common.Label][]*common.Expression)
		for label, expressions := range labels.GetLabelExpressions() {

			if _, knownLabel := labelOrder[common.Label(label)]; !knownLabel {
				return nil, 0, errox.InvalidArgs.CausedByf("unknown label %q for metric %q", label, metric)
			}

			var exprs []*common.Expression
			for _, expr := range expressions.GetExpression() {
				if expr, err := common.MakeExpression(expr.GetOperator(), expr.GetArgument()); err != nil {
					return nil, 0, errox.InvalidArgs.CausedByf(
						"failed to parse expression for metric %q with label %q: %v",
						metric, label, err)
				} else {
					exprs = append(exprs, expr)
				}
			}
			metricLabels[common.Label(label)] = exprs
		}
		result[common.MetricName(metric)] = metricLabels
	}
	if len(result) == 0 {
		return nil, 0, nil
	}
	return result, period, nil
}
