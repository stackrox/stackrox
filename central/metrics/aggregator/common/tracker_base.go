package common

import (
	"context"
	"iter"
	"slices"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var log = logging.CreateLogger(logging.ModuleForName("central_metrics"), 1)

type Tracker interface {
	Run(context.Context)
	Reconfigure(context.Context, *Configuration)
}

type FindingGenerator[Finding Countable] func(context.Context, *v1.Query, MetricsConfiguration) iter.Seq[Finding]

// TrackerBase implements a generic finding tracker.
// Configured with a finding generator and other arguments, it runs a goroutine
// that periodically aggregates gathered values and updates the gauge values.
type TrackerBase[Finding Countable] struct {
	category    string
	description string
	labelOrder  map[Label]int
	getters     map[Label]func(Finding) string
	generator   FindingGenerator[Finding]
	gauge       func(string, prometheus.Labels, int)

	// metricsConfig can be changed with an API call.
	config           *Configuration
	metricsConfigMux sync.RWMutex

	ticker *time.Ticker

	// Mockable for testing purposes:
	registerMetricFunc   func(*Configuration, MetricName)
	unregisterMetricFunc func(MetricName)
}

// makeGettersMap transforms a list of label names with their getters to a map.
func makeGettersMap[Finding Countable](getters []LabelGetter[Finding]) map[Label]func(Finding) string {
	result := make(map[Label]func(Finding) string, len(getters))
	for _, getter := range getters {
		result[getter.Label] = getter.Getter
	}
	return result
}

// getMetricLabels extracts the metric labels from the filter expression and
// sort them according to the labelOrder map values.
// This makes the labels to appear in the stable order in the Prometheus output.
func getMetricLabels(labelExpression map[Label]Expression, labelOrder map[Label]int) []string {
	var labels []string
	for label := range labelExpression {
		labels = append(labels, string(label))
	}
	slices.SortFunc(labels, func(a, b string) int {
		return labelOrder[Label(a)] - labelOrder[Label(b)]
	})
	return labels
}

// MakeTrackerBase initializes a tracker without any period or metrics
// configuration. Call Reconfigure to configure the period and the metrics.
func MakeTrackerBase[Finding Countable](category, description string,
	getters []LabelGetter[Finding], generator FindingGenerator[Finding],
	gauge func(string, prometheus.Labels, int),
) *TrackerBase[Finding] {
	tracker := &TrackerBase[Finding]{
		category:    category,
		description: description,
		labelOrder:  MakeLabelOrderMap(getters),
		getters:     makeGettersMap(getters),
		generator:   generator,
		gauge:       gauge,
		config:      &Configuration{},
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
	// Force track only on reconfiguration, not on the initial tracker creation.
	if previous.period != cfg.period && cfg.period != 0 {
		tracker.track(ctx)
	}
	if cfg.period == 0 {
		log.Debugf("Metrics collection has been disabled for %s", tracker.category)
		for metric := range previous.metrics {
			tracker.unregisterMetricFunc(metric)
		}
		return
	}
	for _, metric := range cfg.toDelete {
		tracker.unregisterMetricFunc(metric)
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

func (tracker *TrackerBase[Finding]) registerMetric(cfg *Configuration, metric MetricName) {
	regCfg := cfg.metricRegistry[metric]
	if err := metrics.RegisterCustomAggregatedMetric(
		string(metric),
		tracker.description,
		cfg.period,
		getMetricLabels(cfg.metrics[metric], tracker.labelOrder),
		regCfg.registry,
		metrics.Exposure(regCfg.exposure)); err != nil {
		log.Errorf("Failed to register %s metric %q: %v", tracker.category, metric, err)
	} else {
		log.Debugf("Registered %s Prometheus metric %q on path /metrics/%s", tracker.category, metric,
			regCfg.registry)
	}
}

// updateTicker initializes, stops or resets the ticker.
func (tracker *TrackerBase[Finding]) updateTicker() {
	tracker.metricsConfigMux.Lock()
	defer tracker.metricsConfigMux.Unlock()
	if tracker.config.period > 0 {
		if tracker.ticker == nil {
			tracker.ticker = time.NewTicker(tracker.config.period)
		} else {
			tracker.ticker.Reset(tracker.config.period)
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
	for finding := range tracker.generator(ctx, cfg.filter, cfg.metrics) {
		aggregator.count(finding)
	}
	for metric, records := range aggregator.result {
		for _, rec := range records {
			tracker.gauge(string(metric), rec.labels, rec.total)
		}
	}
}

func (tracker *TrackerBase[Finding]) Run(ctx context.Context) {
	defer tracker.ticker.Stop()
	tracker.track(ctx)

	for {
		select {
		case <-tracker.ticker.C:
			tracker.track(ctx)
		case <-ctx.Done():
			return
		}
	}
}
