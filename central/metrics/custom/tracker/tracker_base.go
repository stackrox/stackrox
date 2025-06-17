package tracker

import (
	"context"
	"iter"
	"maps"
	"slices"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var log = logging.CreateLogger(logging.ModuleForName("central_metrics"), 1)

// LazyLabel enables deferred evaluation of a label's value.
// Computing and storing values for all labels for every finding would be
// inefficient. Instead, the Getter function computes the value for this
// specific label only when provided with a finding.
type LazyLabel[Finding any] struct {
	Label
	Getter func(*Finding) string
}

// MakeLabelOrderMap maps labels to their order according to the order of
// the labels in the list of getters.
// Respecting the order is important for computing the aggregation key, which is
// a concatenation of label values.
func MakeLabelOrderMap[Finding any](getters []LazyLabel[Finding]) map[Label]int {
	result := make(map[Label]int, len(getters))
	for i, getter := range getters {
		result[getter.Label] = i + 1
	}
	return result
}

type Tracker interface {
	Gather(context.Context)
	Reconfigure(*Configuration)
}

// FindingGenerator returns an iterator to the sequence of findings.
type FindingGenerator[Finding any] func(context.Context, MetricsConfiguration) iter.Seq[*Finding]

type gatherer struct {
	lastGather time.Time
	running    atomic.Bool
	registry   metrics.CustomRegistry
}

// TrackerBase implements a generic finding tracker.
// Configured with a finding generator and other arguments, it runs a goroutine
// that periodically aggregates gathered values and updates the gauge values.
type TrackerBase[Finding any] struct {
	category    string
	description string
	labelOrder  map[Label]int
	getters     map[Label]func(*Finding) string
	generator   FindingGenerator[Finding]

	// metricsConfig can be changed with an API call.
	config           *Configuration
	metricsConfigMux sync.RWMutex

	gatherer *gatherer
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
	registry metrics.CustomRegistry,
) *TrackerBase[Finding] {
	return &TrackerBase[Finding]{
		category:    category,
		description: description,
		labelOrder:  MakeLabelOrderMap(getters),
		getters:     makeGettersMap(getters),
		generator:   generator,
		gatherer:    &gatherer{registry: registry},
	}
}

// Reconfigure assumes the configuration has been validated, so doesn't return
// an error.
func (tracker *TrackerBase[Finding]) Reconfigure(cfg *Configuration) {
	if cfg == nil {
		cfg = &Configuration{}
	}
	previous := tracker.SetConfiguration(cfg)
	if previous != nil {
		if cfg.period == 0 {
			log.Debugf("Metrics collection has been disabled for %s", tracker.category)
			tracker.unregisterMetrics(slices.Collect(maps.Keys(previous.metrics)))
			return
		}
		tracker.unregisterMetrics(cfg.toDelete)
	}
	tracker.registerMetrics(cfg, cfg.toAdd)
}

func labelsAsStrings(labels []Label) []string {
	strings := make([]string, len(labels))
	for i, label := range labels {
		strings[i] = string(label)
	}
	return strings
}

func (tracker *TrackerBase[Finding]) unregisterMetrics(metrics []MetricName) {
	for _, metric := range metrics {
		if tracker.gatherer.registry.UnregisterMetric(string(metric)) {
			log.Debugf("Unregistered %s Prometheus metric %q", tracker.category, metric)
		}
	}
}

func (tracker *TrackerBase[Finding]) registerMetrics(cfg *Configuration, metrics []MetricName) {
	for _, metric := range metrics {
		tracker.registerMetric(tracker.gatherer, cfg, metric)
	}
}

func (tracker *TrackerBase[Finding]) registerMetric(gatherer *gatherer, cfg *Configuration, metric MetricName) {
	if err := gatherer.registry.RegisterMetric(
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
func (tracker *TrackerBase[Finding]) track(ctx context.Context, registry metrics.CustomRegistry, metrics MetricsConfiguration) {
	if len(metrics) == 0 {
		return
	}
	aggregator := makeAggregator(metrics, tracker.labelOrder, tracker.getters)
	for finding := range tracker.generator(ctx, metrics) {
		aggregator.count(finding)
	}
	for metric, records := range aggregator.result {
		for _, rec := range records {
			registry.SetTotal(string(metric), rec.labels, rec.total)
		}
	}
}

// Gather the data not more often then maxAge.
func (tracker *TrackerBase[Finding]) Gather(ctx context.Context) {
	cfg := tracker.GetConfiguration()
	gatherer := tracker.gatherer

	// Return if is still running.
	if !gatherer.running.CompareAndSwap(false, true) {
		return
	}
	defer gatherer.running.Store(false)

	if cfg.period == 0 || time.Since(gatherer.lastGather) < cfg.period {
		return
	}
	tracker.track(ctx, gatherer.registry, cfg.metrics)
	gatherer.lastGather = time.Now()
}
