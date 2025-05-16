package aggregator

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/metrics"
)

type Label string
type metricName string
type metricKey string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true
type metricsConfig map[metricName]map[Label][]*expression

type record struct {
	labels prometheus.Labels
	total  int
}

type result map[metricName]map[metricKey]*record

var (
	Problemetrics = prometheus.NewRegistry()

	descriptions = map[string]string{
		vulnerabilitiesCategory: "discovered CVEs",
	}

	metricNamePattern = regexp.MustCompile("^[a-zA-Z0-9_]+$")
)

func validateMetricName(s string) error {
	if len(s) == 0 {
		return errors.New("empty")
	}
	if !metricNamePattern.MatchString(s) {
		return errors.New("bad characters")
	}
	return nil
}

func registerMetrics(category string, metricsConfig metricsConfig, period time.Duration) {
	for metric, expressions := range metricsConfig {
		metrics.RegisterCustomAggregatedMetric(string(metric), descriptions[category], period,
			getMetricLabels(expressions), Problemetrics)

		log.Infof("Registered %s Prometheus metric %q", category, metric)
	}
}

type trackWrapper[DS any] struct {
	ds         DS
	gatherFunc func(context.Context, DS, metricsConfig) result
	cfgGetter  func() metricsConfig
	trackFunc  func(metricName string, labels prometheus.Labels, total int)
}

func makeTrackWrapper[DS any](ds DS, cfgGetter func() metricsConfig, gatherFunc func(context.Context, DS, metricsConfig) result) *trackWrapper[DS] {
	return &trackWrapper[DS]{
		ds:         ds,
		gatherFunc: gatherFunc,
		cfgGetter:  cfgGetter,
		trackFunc:  metrics.SetCustomAggregatedCount,
	}
}

func (tw *trackWrapper[DS]) track(ctx context.Context) {
	for metric, records := range tw.gatherFunc(ctx, tw.ds, tw.cfgGetter()) {
		for _, rec := range records {
			tw.trackFunc(string(metric), rec.labels, rec.total)
		}
	}
}
