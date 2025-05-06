package telemetry

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	configDS "github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
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

func parseConfig() (map[metricName][]*expression, time.Duration, error) {

	systemPrivateConfig, err := configDS.Singleton().GetPrivateConfig(
		sac.WithAllAccess(context.Background()))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get metrics configuration from the database: %w", err)
	}
	config := systemPrivateConfig.GetPrometheusMetricsConfig()
	if config == nil {
		return nil, 0, nil
	}
	period := time.Hour * time.Duration(config.GetGatheringPeriodHours())
	if period == 0 {
		return nil, period, nil
	}
	result := make(map[metricName][]*expression)
	for _, metric := range config.GetMetrics() {
		if metric == nil {
			continue
		}
		if err := validateMetricName(metric.GetName()); err != nil {
			return nil, 0, errox.InvalidArgs.CausedByf(
				"invalid metric name %q: %v", metric.GetName(), err)
		}
		for _, label := range metric.GetLabels() {
			expr := &expression{
				label: label.GetName(),
				op:    operator(label.GetExpression().GetOperator()),
				arg:   label.GetExpression().GetArgument(),
			}
			if err := expr.validate(); err != nil {
				return nil, 0, errox.InvalidArgs.CausedByf(
					"failed to parse expression for metric %q: %v",
					metric.GetName(), err)
			}
			result[metric.GetName()] = append(result[metric.GetName()], expr)
		}
	}
	if len(result) == 0 {
		return nil, 0, nil
	}
	return result, period, nil
}
