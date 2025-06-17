package common

import (
	"context"
	"iter"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var log = logging.CreateLogger(logging.ModuleForName("central_metrics"), 1)

type LazyLabel[Finding any] struct {
	Label
	Getter func(*Finding) string
}

// MakeLabelOrderMap maps labels to their order according to the order of
// the labels in the list of getters.
func MakeLabelOrderMap[Finding any](getters []LazyLabel[Finding]) map[Label]int {
	result := make(map[Label]int, len(getters))
	for i, getter := range getters {
		result[getter.Label] = i + 1
	}
	return result
}

type Tracker interface {
	Run(context.Context)
	Reconfigure(context.Context, *Configuration)
}

type FindingGenerator[Finding any] func(context.Context, MetricsConfiguration) iter.Seq[*Finding]

// TrackerBase implements a generic finding tracker.
// Configured with a finding generator and other arguments, it runs a goroutine
// that periodically aggregates gathered values and updates the gauge values.
type TrackerBase[Finding any] struct {
	category    string
	description string
	labelOrder  map[Label]int
	getters     map[Label]func(*Finding) string
	generator   FindingGenerator[Finding]
	gauge       func(string, prometheus.Labels, int)

	// metricsConfig can be changed with an API call.
	config           *Configuration
	metricsConfigMux sync.RWMutex

	ticker  *time.Ticker
	running atomic.Bool

	// Mockable for testing purposes:
	registerMetricFunc   func(*Configuration, MetricName)
	unregisterMetricFunc func(MetricName)
}

// makeGettersMap transforms a list of label names with their getters to a map.
func makeGettersMap[Finding any](getters []LazyLabel[Finding]) map[Label]func(*Finding) string {
	result := make(map[Label]func(*Finding) string, len(getters))
	for _, getter := range getters {
		result[getter.Label] = getter.Getter
	}
	return result
}

// MakeTrackerBase initializes a tracker without any period or metrics
// configuration. Call Reconfigure to configure the period and the metrics.
func MakeTrackerBase[Finding any](category, description string,
	getters []LazyLabel[Finding], generator FindingGenerator[Finding],
	gauge func(string, prometheus.Labels, int),
) *TrackerBase[Finding] {
	tracker := &TrackerBase[Finding]{
		category:    category,
		description: description,
		labelOrder:  MakeLabelOrderMap(getters),
		getters:     makeGettersMap(getters),
		generator:   generator,
		gauge:       gauge,
	}
	tracker.registerMetricFunc = tracker.registerMetric
	tracker.unregisterMetricFunc = tracker.unregisterMetric
	return tracker
}

// Reconfigure assumes the configuration has been validated, so doesn't return
// an error.
func (tracker *TrackerBase[Finding]) Reconfigure(ctx context.Context, cfg *Configuration) {
	if cfg == nil {
		cfg = &Configuration{}
	}
	previous := tracker.SetConfiguration(cfg)
	tracker.updateTicker()
	if previous != nil {
		if cfg.period != 0 {
			// Force track only on reconfiguration, not on the initial tracker
			// creation.
			if tracker.running.Load() {
				tracker.track(ctx)
			} else {
				go tracker.Run(ctx)
			}
		} else {
			log.Debugf("Metrics collection has been disabled for %s", tracker.category)
			for metric := range previous.metrics {
				tracker.unregisterMetricFunc(metric)
			}
			return
		}
		for _, metric := range cfg.toDelete {
			tracker.unregisterMetricFunc(metric)
		}
	}
	for _, metric := range cfg.toAdd {
		tracker.registerMetricFunc(cfg, metric)
	}
}

func (tracker *TrackerBase[Finding]) unregisterMetric(metric MetricName) {
	if metrics.UnregisterCustomAggregatedMetric(string(metric)) {
		log.Debugf("Unregistered %s Prometheus metric %q", tracker.category, metric)
	}
}

func labelsAsStrings(labels []Label) []string {
	strings := make([]string, len(labels))
	for i, label := range labels {
		strings[i] = string(label)
	}
	return strings
}

func (tracker *TrackerBase[Finding]) registerMetric(cfg *Configuration, metric MetricName) {
	if err := metrics.RegisterCustomAggregatedMetric(
		string(metric),
		tracker.description,
		cfg.period,
		labelsAsStrings(cfg.metrics[metric]),
	); err != nil {
		log.Errorf("Failed to register %s metric %q: %v", tracker.category, metric, err)
		return
	}
	log.Debugf("Registered %s Prometheus metric %q", tracker.category, metric)
}

// updateTicker initializes, stops or resets the ticker.
func (tracker *TrackerBase[Finding]) updateTicker() {
	tracker.metricsConfigMux.Lock()
	defer tracker.metricsConfigMux.Unlock()
	period := tracker.config.period
	if period > 0 {
		if tracker.ticker == nil {
			tracker.ticker = time.NewTicker(period)
		} else {
			tracker.ticker.Reset(period)
		}
		return
	}
	if tracker.ticker != nil {
		tracker.ticker.Stop()
	}
}

func (tracker *TrackerBase[Finding]) GetConfiguration() *Configuration {
	tracker.metricsConfigMux.RLock()
	defer tracker.metricsConfigMux.RUnlock()
	return tracker.config
}

func (tracker *TrackerBase[Finding]) SetConfiguration(config *Configuration) *Configuration {
	tracker.metricsConfigMux.Lock()
	defer tracker.metricsConfigMux.Unlock()
	previous := tracker.config
	tracker.config = config
	return previous
}

// track aggregates the fetched findings and updates the gauges.
func (tracker *TrackerBase[Finding]) track(ctx context.Context) {
	cfg := tracker.GetConfiguration()
	if len(cfg.metrics) == 0 {
		return
	}
	aggregator := makeAggregator(cfg.metrics, tracker.labelOrder, tracker.getters)
	for finding := range tracker.generator(ctx, cfg.metrics) {
		aggregator.count(finding)
	}
	for metric, records := range aggregator.result {
		for _, rec := range records {
			tracker.gauge(string(metric), rec.labels, rec.total)
		}
	}
}

func (tracker *TrackerBase[Finding]) getTicker() *time.Ticker {
	tracker.metricsConfigMux.RLock()
	defer tracker.metricsConfigMux.RUnlock()
	return tracker.ticker
}

// Run starts the gathering loop. It can be called explicitly, or via
// non-initial reconfiguration from zero period to non-zero period.
func (tracker *TrackerBase[Finding]) Run(ctx context.Context) {
	ticker := tracker.getTicker()

	// Return if no ticker or is already running.
	if ticker == nil || !tracker.running.CompareAndSwap(false, true) {
		return
	}
	defer tracker.running.Store(false)
	defer ticker.Stop()

	tracker.track(ctx)

	for {
		select {
		case <-ticker.C:
			tracker.track(ctx)
		case <-ctx.Done():
			return
		}
	}
}
