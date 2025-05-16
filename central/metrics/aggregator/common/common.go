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
	log = logging.LoggerForModule()

	metricNamePattern = regexp.MustCompile("^[a-zA-Z0-9_]+$")
)

type Label string      // Prometheus label.
type MetricName string // Prometheus metric name.
type MetricKey string  // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true

// MetricLabelExpressions is the parsed aggregation configuration.
type MetricLabelExpressions map[MetricName]map[Label][]*Expression

// Result is the aggregation result.
type Result map[MetricName]map[MetricKey]*Record

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

// ValidateMetricName ensures the name is alnum_.
func ValidateMetricName(name string) error {
	if len(name) == 0 {
		return errors.New("empty")
	}
	if !metricNamePattern.MatchString(name) {
		return errors.New("bad characters")
	}
	return nil
}

func registerMetrics(registry *prometheus.Registry, category string, description string, labelOrder map[Label]int, mle MetricLabelExpressions, period time.Duration) {
	for metric, labelExpressions := range mle {
		metrics.RegisterCustomAggregatedMetric(string(metric), description, period,
			getMetricLabels(labelExpressions, labelOrder), registry)

		log.Infof("Registered %s Prometheus metric %q", category, metric)
	}
}

// MakeTrackFunc calls trackFunc on every metric returned by gatherFunc.
// cfgGetter returns the current configuration, which may dynamically change.
func MakeTrackFunc[DS any](
	ds DS,
	cfgGetter func() MetricLabelExpressions,
	gatherFunc func(context.Context, DS, MetricLabelExpressions) Result,
	trackFunc func(metricName string, labels prometheus.Labels, total int),
) func(context.Context) {

	return func(ctx context.Context) {
		for metric, records := range gatherFunc(ctx, ds, cfgGetter()) {
			for _, rec := range records {
				trackFunc(string(metric), rec.labels, rec.total)
			}
		}
	}
}
