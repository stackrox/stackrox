package common

import (
	"context"
	"fmt"
	"strings"
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

	ticker *time.Ticker
}

type Tracker interface {
	Run(context.Context)
	Reconfigure(context.Context, string, map[string]*storage.PrometheusMetricsConfig_Labels, time.Duration) error
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
	}
}

func (tc *TrackerConfig[Finding]) Reconfigure(ctx context.Context, filter string, cfg map[string]*storage.PrometheusMetricsConfig_Labels, period time.Duration) error {
	for metric, labels := range cfg {
		if err := metrics.CheckExposureChange(metric, labels.GetRegistryName(), metrics.Exposure(labels.GetExposure())); err != nil {
			return errInvalidConfiguration.CausedBy(err)
		}
	}

	mcfg, err := parseMetricLabels(cfg, tc.labelOrder)
	if err != nil {
		return err
	}

	toAdd, toDelete, changed := tc.metricsConfig.DiffLabels(mcfg)
	if len(changed) != 0 {
		return errInvalidConfiguration.CausedByf("cannot alter metrics %v", changed)
	}
	if len(toAdd) == 0 && len(toDelete) == 0 {
		return nil
	}

	q, err := search.ParseQuery(filter, search.MatchAllIfEmpty())
	if err != nil {
		return errInvalidConfiguration.CausedBy(err)
	}
	tc.SetMetricsConfiguration(q, mcfg)
	// Force track only on reconfiguration, not on the initial tracker creation.
	if tc.setPeriod(period) {
		tc.track(ctx)
	}

	if period == 0 {
		log.Infof("Metrics collection is disabled for %s", tc.category)
	}

	for _, metric := range toDelete {
		if metrics.UnregisterCustomAggregatedMetric(string(metric)) {
			regCfg := cfg[string(metric)]
			log.Infof("Unregistered %s Prometheus metric %q from path %s", tc.category, metric,
				strings.Join([]string{"/metrics", regCfg.GetRegistryName()}, "/"))
		}
	}
	for _, metric := range toAdd {
		regCfg := cfg[string(metric)]
		if err := metrics.RegisterCustomAggregatedMetric(
			string(metric),
			tc.description,
			period,
			getMetricLabels(mcfg[metric], tc.labelOrder),
			regCfg.GetRegistryName(),
			metrics.Exposure(regCfg.GetExposure().Number())); err != nil {
			return fmt.Errorf("failed to register %s metric %q: %w", tc.category, metric, err)
		}
		if period > 0 {
			log.Infof("Registered %s Prometheus metric %q on path %s", tc.category, metric,
				strings.Join([]string{"/metrics", regCfg.GetRegistryName()}, "/"))
		}
	}
	return nil
}

// setPeriod returns true if tracker period has been reconfigured.
func (tc *TrackerConfig[Finding]) setPeriod(period time.Duration) bool {
	tc.metricsConfigMux.Lock()
	defer tc.metricsConfigMux.Unlock()
	if period > 0 {
		if tc.ticker == nil {
			tc.ticker = time.NewTicker(period)
			return false
		}
		tc.ticker.Reset(period)
		return true
	}
	if tc.ticker != nil {
		tc.ticker.Stop()
	}
	return false
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

func (tc *TrackerConfig[Finding]) track(ctx context.Context) {
	query, mcfg := tc.GetMetricsConfiguration()
	if len(mcfg) == 0 {
		return
	}
	aggregator := makeAggregator(mcfg, tc.labelOrder)
	for finding := range tc.generator(ctx, query, mcfg) {
		aggregator.count(func(label Label) string {
			return tc.getters[label](finding)
		}, finding.Count())
	}
	for metric, records := range aggregator.result {
		for _, rec := range records {
			tc.gauge(string(metric), rec.labels, rec.total)
		}
	}
}

func (tc *TrackerConfig[Finding]) Run(ctx context.Context) {
	defer tc.ticker.Stop()
	tc.track(ctx)

	for {
		select {
		case <-tc.ticker.C:
			tc.track(ctx)
		case <-ctx.Done():
			return
		}
	}
}
