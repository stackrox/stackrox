package common

import (
	"maps"
	"regexp"
	"slices"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/search"
)

var (
	errInvalidConfiguration = errox.InvalidArgs.New("invalid configuration")

	// Source: https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
	metricNamePattern = regexp.MustCompile("^[a-zA-Z_:][a-zA-Z0-9_:]*$")

	registryNamePattern = regexp.MustCompile("^[a-zA-Z0-9-_]*$")
)

type metricExposure struct {
	registry string
	exposure metrics.Exposure
}

type Configuration struct {
	metrics        MetricsConfiguration
	metricRegistry map[MetricName]metricExposure
	toAdd          []MetricName
	toDelete       []MetricName
	filter         *v1.Query
	period         time.Duration
}

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

// parseMetricLabels converts the storage object to the usable map, validating
// the values.
func parseMetricLabels(config map[string]*storage.PrometheusMetricsConfig_Labels, labelOrder map[Label]int) (MetricsConfiguration, error) {
	result := make(MetricsConfiguration, len(config))
	for metric, labels := range config {

		if !registryNamePattern.MatchString(labels.GetRegistryName()) {
			return nil, errInvalidConfiguration.CausedByf(
				`registry name %q for metric %s doesn't match "`+registryNamePattern.String()+`"`, labels.GetRegistryName(), metric)
		}

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

func ParseConfiguration(cfg *storage.PrometheusMetricsConfig_Metrics, currentMetrics MetricsConfiguration, labelOrder map[Label]int) (*Configuration, error) {
	metricRegistry := make(map[MetricName]metricExposure, len(cfg.GetMetrics()))

	for metric, labels := range cfg.GetMetrics() {
		if err := metrics.CheckExposureChange(metric,
			labels.GetRegistryName(),
			metrics.Exposure(labels.GetExposure())); err != nil {
			return nil, errInvalidConfiguration.CausedBy(err)
		}

		metricRegistry[MetricName(metric)] = metricExposure{
			labels.GetRegistryName(),
			metrics.Exposure(labels.GetExposure())}
	}

	mcfg, err := parseMetricLabels(cfg.GetMetrics(), labelOrder)
	if err != nil {
		return nil, err
	}
	toAdd, toDelete, changed := currentMetrics.DiffLabels(mcfg)
	if len(changed) != 0 {
		return nil, errInvalidConfiguration.CausedByf("cannot alter metrics %v", changed)
	}
	q, err := search.ParseQuery(cfg.GetFilter(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errInvalidConfiguration.CausedBy(err)
	}
	return &Configuration{
		metrics:        mcfg,
		metricRegistry: metricRegistry,
		toAdd:          toAdd,
		toDelete:       toDelete,
		filter:         q,
		period:         time.Minute * time.Duration(cfg.GetGatheringPeriodMinutes()),
	}, nil
}
