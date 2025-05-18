package common

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
)

// TrackerConfig wraps various pieces of configuration required for tracking
// various metrics.
type TrackerConfig struct {
	category    string
	description string
	labelOrder  map[Label]int
	generator   FindingGenerator

	// metricsConfig can be changed with an API call.
	metricsConfig    MetricLabelExpressions
	metricsConfigMux sync.RWMutex

	// periodCh allows for changing the period in runtime.
	periodCh chan time.Duration
}

// MakeTrackerConfig initializes a tracker configuration without any period or metric expressions.
// Call Reconfigure to configure the period and the expressions.
func MakeTrackerConfig(category, description string, labelOrder map[Label]int, generator FindingGenerator) *TrackerConfig {
	return &TrackerConfig{
		category:    category,
		description: description,
		labelOrder:  labelOrder,
		generator:   generator,

		periodCh: make(chan time.Duration, 1),
	}
}

func (tc *TrackerConfig) GetPeriodCh() <-chan time.Duration {
	return tc.periodCh
}

func (tc *TrackerConfig) Reconfigure(registry *prometheus.Registry, cfg map[string]*storage.PrometheusMetricsConfig_LabelExpressions, period time.Duration) error {
	mle, err := parseMetricLabels(cfg, tc.labelOrder)
	if err != nil {
		return err
	}
	tc.SetMetricLabelExpressions(mle)
	select {
	case tc.periodCh <- period:
		break
	default:
		// If the period has not been read, read it now:
		<-tc.periodCh
		tc.periodCh <- period
	}
	tc.registerMetrics(registry, period)
	return nil
}

func (tc *TrackerConfig) GetMetricLabelExpressions() MetricLabelExpressions {
	tc.metricsConfigMux.RLock()
	defer tc.metricsConfigMux.RUnlock()
	return tc.metricsConfig
}

func (tc *TrackerConfig) SetMetricLabelExpressions(mle MetricLabelExpressions) {
	tc.metricsConfigMux.Lock()
	defer tc.metricsConfigMux.Unlock()
	tc.metricsConfig = mle
}

func (tc *TrackerConfig) registerMetrics(registry *prometheus.Registry, period time.Duration) {
	if period == 0 {
		log.Infof("Metrics collection is disabled for %s", tc.category)
	}
	for metric, labelExpressions := range tc.metricsConfig {
		metrics.RegisterCustomAggregatedMetric(string(metric), tc.description, period,
			getMetricLabels(labelExpressions, tc.labelOrder), registry)

		log.Infof("Registered %s Prometheus metric %q", tc.category, metric)
	}
}

// MakeTrackFunc returns a function that calls trackFunc on every metric
// returned by gatherFunc. cfgGetter returns the current configuration, which
// may dynamically change.
func MakeTrackFunc(
	cfg *TrackerConfig,
	trackFunc func(metricName string, labels prometheus.Labels, total int),
) func(context.Context) {

	return func(ctx context.Context) {
		result := makeResult(cfg.GetMetricLabelExpressions(), cfg.labelOrder)
		for finding := range cfg.generator(ctx) {
			result.count(finding)
		}
		for metric, records := range result.aggregated {
			for _, rec := range records {
				trackFunc(string(metric), rec.labels, rec.total)
			}
		}
	}
}
