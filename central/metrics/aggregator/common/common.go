package common

import (
	"context"
	"errors"
	"iter"
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

type Label string               // Prometheus label.
type MetricName string          // Prometheus metric name.
type Finding func(Label) string // Lazy map.
type FindingIterator func(context.Context) iter.Seq[Finding]

// MetricLabelExpressions is the parsed aggregation configuration.
type MetricLabelExpressions map[MetricName]map[Label][]*Expression

type metricKey string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true

// result is the aggregation result.
type result struct {
	aggregated map[MetricName]map[metricKey]*record
	mle        MetricLabelExpressions
	labelOrder map[Label]int
}

func makeResult(mle MetricLabelExpressions, labelOrder map[Label]int) *result {
	aggregated := make(map[MetricName]map[metricKey]*record)
	for metric := range mle {
		aggregated[metric] = make(map[metricKey]*record)
	}
	return &result{aggregated, mle, labelOrder}
}

func (r *result) count(finding Finding) {
	for metric, expressions := range r.mle {
		if key, labels := makeAggregationKeyInstance(expressions, finding, r.labelOrder); key != "" {
			if rec, ok := r.aggregated[metric][key]; ok {
				rec.Inc()
			} else {
				r.aggregated[metric][key] = &record{labels, 1}
			}
		}
	}
}

// record is a single gauge metric record.
type record struct {
	labels prometheus.Labels
	total  int
}

func (r *record) Inc() {
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
func MakeTrackFunc(
	cfg *TrackerConfig,
	cfgGetter func() MetricLabelExpressions,
	trackFunc func(metricName string, labels prometheus.Labels, total int),
) func(context.Context) {

	return func(ctx context.Context) {
		result := makeResult(cfgGetter(), cfg.labelOrder)
		for finding := range cfg.gather(ctx) {
			result.count(finding)
		}
		for metric, records := range result.aggregated {
			for _, rec := range records {
				trackFunc(string(metric), rec.labels, rec.total)
			}
		}
	}
}
