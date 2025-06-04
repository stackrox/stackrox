package common

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

// TrackerConfig wraps various pieces of configuration required for tracking
// various metrics.
type TrackerConfig[Finding Countable] struct {
	category    string
	description string
	labelOrder  map[Label]int
	getters     map[Label]func(Finding) string
	generator   FindingGenerator[Finding]
	gauge       func(string, prometheus.Labels, int)

	// metricsConfig can be changed with an API call.
	metricsConfig    MetricsConfiguration
	metricsConfigMux sync.RWMutex
	query            *v1.Query

	// periodCh allows for changing the period in runtime.
	periodCh chan time.Duration
	sync.Once
}

type Tracker interface {
	Do(func())
	GetPeriodCh() chan time.Duration
	Track(context.Context)
	Reconfigure(*prometheus.Registry, string, map[string]*storage.PrometheusMetricsConfig_Labels, time.Duration) error
}

func makeLabelOrderMap[Finding Countable](getters []LabelGetter[Finding]) map[Label]int {
	result := make(map[Label]int, len(getters))
	for i, getter := range getters {
		result[getter.Label] = i + 1
	}
	return result
}

func makeGettersMap[Finding Countable](getters []LabelGetter[Finding]) map[Label]func(Finding) string {
	result := make(map[Label]func(Finding) string, len(getters))
	for _, getter := range getters {
		result[getter.Label] = getter.Getter
	}
	return result
}

// MakeTrackerConfig initializes a tracker configuration without any period or metrics configuration.
// Call Reconfigure to configure the period and the metrics.
func MakeTrackerConfig[Finding Countable](category, description string,
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

func (tc *TrackerConfig[Finding]) Reconfigure(registry *prometheus.Registry, filter string, cfg map[string]*storage.PrometheusMetricsConfig_Labels, period time.Duration) error {
	mcfg, err := parseMetricLabels(cfg, tc.labelOrder)
	if err != nil {
		return err
	}
	q, err := search.ParseQuery(filter, search.MatchAllIfEmpty())
	if err != nil {
		return err
	}
	tc.SetMetricsConfiguration(q, mcfg)
	select {
	case tc.periodCh <- period:
	default:
		// The period should be read from the channel by the runner loop.
		// If the channel buffer is full and so the channel is blocked for writing,
		// purge it now and send the new value to proceed with the reconfiguration:
		select {
		case <-tc.periodCh:
		default:
			tc.periodCh <- period
		}
	}

	if period == 0 {
		log.Infof("Metrics collection is disabled for %s", tc.category)
		return nil
	}

	return tc.registerMetrics(registry, period)
}

func (tc *TrackerConfig[Finding]) GetMetricsConfiguration() (*v1.Query, MetricsConfiguration) {
	tc.metricsConfigMux.RLock()
	defer tc.metricsConfigMux.RUnlock()
	return tc.query, tc.metricsConfig
}

func (tc *TrackerConfig[Finding]) SetMetricsConfiguration(query *v1.Query, mcfg MetricsConfiguration) {
	tc.metricsConfigMux.Lock()
	defer tc.metricsConfigMux.Unlock()
	tc.query = query
	tc.metricsConfig = mcfg
}

func (tc *TrackerConfig[Finding]) registerMetrics(registry *prometheus.Registry, period time.Duration) error {
	for metric, labelExpression := range tc.metricsConfig {
		if err := metrics.RegisterCustomAggregatedMetric(string(metric), tc.description, period,
			getMetricLabels(labelExpression, tc.labelOrder), registry); err != nil {
			return fmt.Errorf("failed to register %s metric %q: %w", tc.category, metric, err)
		}
		log.Infof("Registered %s Prometheus metric %q", tc.category, metric)
	}
	return nil
}

// MakeTrackFunc returns a function that calls trackFunc on every metric
// returned by gatherFunc. cfgGetter returns the current configuration, which
// may dynamically change.
func (cfg *TrackerConfig[Finding]) Track(ctx context.Context) {
	query, mcfg := cfg.GetMetricsConfiguration()
	if len(mcfg) == 0 {
		return
	}
	aggregator := makeAggregator(mcfg, cfg.labelOrder)
	for finding := range cfg.generator(ctx, query, mcfg) {
		aggregator.count(func(label Label) string {
			return cfg.getters[label](finding)
		}, finding.Count())
	}
	for metric, records := range aggregator.result {
		for _, rec := range records {
			cfg.gauge(string(metric), rec.labels, rec.total)
		}
	}
}
