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

type Label string
type MetricName string
type MetricKey string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true
type MetricsConfig map[MetricName]map[Label][]*Expression

type Record struct {
	labels prometheus.Labels
	total  int
}

func MakeRecord(labels prometheus.Labels, total int) *Record {
	return &Record{labels, total}
}

func (r *Record) Inc() {
	r.total++
}

type Result map[MetricName]map[MetricKey]*Record

var (
	Problemetrics = prometheus.NewRegistry()

	log = logging.LoggerForModule()

	metricNamePattern = regexp.MustCompile("^[a-zA-Z0-9_]+$")
)

func ValidateMetricName(s string) error {
	if len(s) == 0 {
		return errors.New("empty")
	}
	if !metricNamePattern.MatchString(s) {
		return errors.New("bad characters")
	}
	return nil
}

func registerMetrics(category string, description string, labelOrder map[Label]int, metricsConfig MetricsConfig, period time.Duration) {
	for metric, expressions := range metricsConfig {
		metrics.RegisterCustomAggregatedMetric(string(metric), description, period,
			getMetricLabels(expressions, labelOrder), Problemetrics)

		log.Infof("Registered %s Prometheus metric %q", category, metric)
	}
}

type TrackWrapper[DS any] struct {
	ds         DS
	gatherFunc func(context.Context, DS, MetricsConfig) Result
	cfgGetter  func() MetricsConfig
	TrackFunc  func(metricName string, labels prometheus.Labels, total int)
}

func MakeTrackWrapper[DS any](ds DS, cfgGetter func() MetricsConfig, gatherFunc func(context.Context, DS, MetricsConfig) Result) *TrackWrapper[DS] {
	return &TrackWrapper[DS]{
		ds:         ds,
		gatherFunc: gatherFunc,
		cfgGetter:  cfgGetter,
		TrackFunc:  metrics.SetCustomAggregatedCount,
	}
}

func (tw *TrackWrapper[DS]) Track(ctx context.Context) {
	for metric, records := range tw.gatherFunc(ctx, tw.ds, tw.cfgGetter()) {
		for _, rec := range records {
			tw.TrackFunc(string(metric), rec.labels, rec.total)
		}
	}
}
