package common

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

// TrackerConfig wraps various pieces of configuration required for tracking
// various metrics.
type TrackerConfig[Finding any] struct {
	category    string
	description string
	labelOrder  map[Label]int
	getters     map[Label]func(Finding) string
	generator   FindingGenerator[Finding]
	gauge       func(string, prometheus.Labels, int)

	// metricsConfig can be changed with an API call.
	metricsConfig    MetricLabelsExpressions
	metricsConfigMux sync.RWMutex

	// periodCh allows for changing the period in runtime.
	periodCh chan time.Duration
	sync.Once
}

type Tracker interface {
	Do(func())
	GetPeriodCh() chan time.Duration
	Track(context.Context)
	Reconfigure(*prometheus.Registry, map[string]*storage.PrometheusMetricsConfig_MetricLabels, time.Duration) error
}

func makeLabelOrderMap[Finding any](getters []LabelGetter[Finding]) map[Label]int {
	result := make(map[Label]int, len(getters))
	for i, getter := range getters {
		result[getter.Label] = i + 1
	}
	return result
}

func makeGettersMap[Finding any](getters []LabelGetter[Finding]) map[Label]func(Finding) string {
	result := make(map[Label]func(Finding) string, len(getters))
	for _, getter := range getters {
		result[getter.Label] = getter.Getter
	}
	return result
}

// MakeTrackerConfig initializes a tracker configuration without any period or metric expressions.
// Call Reconfigure to configure the period and the expressions.
func MakeTrackerConfig[Finding any](category, description string,
	getters []LabelGetter[Finding], generator FindingGenerator[Finding],
	gauge func(string, prometheus.Labels, int),
) *TrackerConfig[Finding] {
	return &TrackerConfig[Finding]{
		category:    category,
		description: description,
		labelOrder:  makeLabelOrderMap(getters),
		getters:     makeGettersMap(getters),
		generator:   generator,
		gauge:       gauge,

		periodCh: make(chan time.Duration, 1),
	}
}

func (tc *TrackerConfig[Finding]) GetPeriodCh() chan time.Duration {
	return tc.periodCh
}

func (tc *TrackerConfig[Finding]) Reconfigure(registry *prometheus.Registry, cfg map[string]*storage.PrometheusMetricsConfig_MetricLabels, period time.Duration) error {
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

	if period == 0 {
		log.Infof("Metrics collection is disabled for %s", tc.category)
		return nil
	}

	return tc.registerMetrics(registry, period)
}

func (tc *TrackerConfig[Finding]) GetMetricLabelExpressions() MetricLabelsExpressions {
	tc.metricsConfigMux.RLock()
	defer tc.metricsConfigMux.RUnlock()
	return tc.metricsConfig
}

func (tc *TrackerConfig[Finding]) SetMetricLabelExpressions(mle MetricLabelsExpressions) {
	tc.metricsConfigMux.Lock()
	defer tc.metricsConfigMux.Unlock()
	tc.metricsConfig = mle
}

func (tc *TrackerConfig[Finding]) registerMetrics(registry *prometheus.Registry, period time.Duration) error {
	for metric, labelExpressions := range tc.metricsConfig {
		if err := metrics.RegisterCustomAggregatedMetric(string(metric), tc.description, period,
			getMetricLabels(labelExpressions, tc.labelOrder), registry); err != nil {
			log.Errorw("Failed to register metrics", logging.Err(err))
			return err
		}
		log.Infof("Registered %s Prometheus metric %q", tc.category, metric)
	}
	return nil
}

// MakeTrackFunc returns a function that calls trackFunc on every metric
// returned by gatherFunc. cfgGetter returns the current configuration, which
// may dynamically change.
func (cfg *TrackerConfig[Finding]) Track(ctx context.Context) {
	mle := cfg.GetMetricLabelExpressions()
	aggregator := makeAggregator(mle, cfg.labelOrder)
	for finding := range cfg.generator(ctx, mle) {
		aggregator.count(func(label Label) string {
			return cfg.getters[label](finding)
		})
	}
	for metric, records := range aggregator.result {
		for _, rec := range records {
			cfg.gauge(string(metric), rec.labels, rec.total)
		}
	}
}
