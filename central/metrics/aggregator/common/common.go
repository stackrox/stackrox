package common

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.CreateLogger(logging.ModuleForName("central_metrics"), 1)

	metricNamePattern = regexp.MustCompile("^[a-zA-Z0-9_]+$")
)

type Label string      // Prometheus label.
type MetricName string // Prometheus metric name.

// MetricLabelExpressions is the parsed aggregation configuration.
type MetricLabelExpressions map[MetricName]map[Label][]*Expression

type metricKey string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true

// Result is the aggregation result.
type Result struct {
	aggregated map[MetricName]map[metricKey]*Record
	mc         MetricLabelExpressions
	labelOrder map[Label]int
}

func MakeResult(mle MetricLabelExpressions, labelOrder map[Label]int) *Result {
	aggregated := make(map[MetricName]map[metricKey]*Record)
	for metric := range mle {
		aggregated[metric] = make(map[metricKey]*Record)
	}
	return &Result{aggregated, mle, labelOrder}
}

func (r *Result) Count(labelGetter func(Label) string) {
	for metric, expressions := range r.mc {
		if key, labels := MakeAggregationKeyInstance(expressions, labelGetter, r.labelOrder); key != "" {
			if rec, ok := r.aggregated[metric][key]; ok {
				rec.Inc()
			} else {
				r.aggregated[metric][key] = MakeRecord(labels, 1)
			}
		}
	}
}

// Record is a single gauge metric record.
type Record struct {
	labels prometheus.Labels
	total  int
}

// MakeRecord contructs a Record instance.
func MakeRecord(labels prometheus.Labels, total int) *Record {
	return &Record{labels, total}
}

func (r *Record) Inc() {
	r.total++
}

// validateMetricName ensures the name is alnum_.
func validateMetricName(name string) error {
	if len(name) == 0 {
		return errors.New("empty")
	}
	if !metricNamePattern.MatchString(name) {
		return errors.New("bad characters")
	}
	return nil
}

func registerMetrics(registry *prometheus.Registry, category string, description string, labelOrder map[Label]int, mle MetricLabelExpressions, period time.Duration) {
	if period == 0 {
		log.Infof("Metrics collection is disabled for %s", category)
	}
	for metric, labelExpressions := range mle {
		metrics.RegisterCustomAggregatedMetric(string(metric), description, period,
			getMetricLabels(labelExpressions, labelOrder), registry)

		log.Infof("Registered %s Prometheus metric %q", category, metric)
	}
}

// MakeTrackFunc returns a function that calls trackFunc on every metric
// returned by gatherFunc. cfgGetter returns the current configuration, which
// may dynamically change.
func MakeTrackFunc[DS any](
	ds DS,
	cfgGetter func() MetricLabelExpressions,
	gatherFunc func(context.Context, DS, MetricLabelExpressions) *Result,
	trackFunc func(metricName string, labels prometheus.Labels, total int),
) func(context.Context) {

	return func(ctx context.Context) {
		for metric, records := range gatherFunc(ctx, ds, cfgGetter()).aggregated {
			for _, rec := range records {
				trackFunc(string(metric), rec.labels, rec.total)
			}
		}
	}
}
