package tracker

import (
	"context"
	"iter"
	"maps"
	"net/http"
	"slices"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

const inactiveGathererTTL = 2 * 24 * time.Hour

var (
	log             = logging.CreateLogger(logging.ModuleForName("central_metrics"), 1)
	ErrStopIterator = errors.New("stopped")
)

type Finding interface {
	GetError() error
	GetIncrement() int
}

type CommonFinding struct{}

func (f *CommonFinding) GetError() error {
	return nil
}

func (f *CommonFinding) GetIncrement() int {
	return 1
}

// LazyLabel enables deferred evaluation of a label's value.
// Computing and storing values for all labels for every finding would be
// inefficient. Instead, the Getter function computes the value for this
// specific label only when provided with a finding.
type LazyLabel[F Finding] struct {
	Label
	Getter func(F) string
}

// MakeLabelOrderMap maps labels to their order according to the order of
// the labels in the list of getters.
// Respecting the order is important for computing the aggregation key, which is
// a concatenation of label values.
func MakeLabelOrderMap[F Finding](getters []LazyLabel[F]) map[Label]int {
	result := make(map[Label]int, len(getters))
	for i, getter := range getters {
		result[getter.Label] = i + 1
	}
	return result
}

// GetLabels returns a slice of labels from the list of lazy getters.
func GetLabels[F Finding](getters []LazyLabel[F]) []string {
	result := make([]string, 0, len(getters))
	for _, l := range getters {
		result = append(result, string(l.Label))
	}
	return result
}

type Tracker interface {
	// Gather the data and update the metrics registry.
	Gather(context.Context)
	// NewConfiguration checks the provided metrics storage configuration
	// and returns a tracker configuration, without reconfiguring the tracker.
	NewConfiguration(*storage.PrometheusMetrics_Group) (*Configuration, error)
	// Reconfigure the tracker with the provided tracker configuration.
	Reconfigure(*Configuration)
}

// FindingGenerator returns an iterator to the sequence of findings.
type FindingGenerator[F Finding] func(context.Context, MetricDescriptors) iter.Seq[F]

type gatherer struct {
	http.Handler
	lastGather time.Time
	running    atomic.Bool
	registry   metrics.CustomRegistry
}

// TrackerBase implements a generic finding tracker.
// Configured with a finding generator and other arguments, it runs a goroutine
// that periodically aggregates gathered values and updates the gauge values.
type TrackerBase[F Finding] struct {
	metricPrefix string
	description  string
	labelOrder   map[Label]int
	getters      map[Label]func(F) string
	generator    FindingGenerator[F]

	// metricsConfig can be changed with an API call.
	config           *Configuration
	metricsConfigMux sync.RWMutex

	gatherers sync.Map       // map[user ID]*gatherer
	cleanupWG sync.WaitGroup // for sync in testing.

	registryFactory func(userID string) metrics.CustomRegistry // for mocking in tests.
}

// makeGettersMap transforms a list of label names with their getters to a map.
func makeGettersMap[F Finding](getters []LazyLabel[F]) map[Label]func(F) string {
	result := make(map[Label]func(F) string, len(getters))
	for _, getter := range getters {
		result[getter.Label] = getter.Getter
	}
	return result
}

// MakeTrackerBase initializes a tracker without any period or metrics
// configuration. Call Reconfigure to configure the period and the metrics.
func MakeTrackerBase[F Finding](metricPrefix, description string,
	getters []LazyLabel[F], generator FindingGenerator[F],
) *TrackerBase[F] {
	return &TrackerBase[F]{
		metricPrefix:    metricPrefix,
		description:     description,
		labelOrder:      MakeLabelOrderMap(getters),
		getters:         makeGettersMap(getters),
		generator:       generator,
		registryFactory: metrics.GetCustomRegistry,
	}
}

// NewConfiguration does not apply the configuration.
func (tracker *TrackerBase[Finding]) NewConfiguration(cfg *storage.PrometheusMetrics_Group) (*Configuration, error) {
	current := tracker.getConfiguration()
	if current == nil {
		current = &Configuration{}
	}

	md, err := translateStorageConfiguration(cfg.GetDescriptors(), tracker.metricPrefix, tracker.labelOrder)
	if err != nil {
		return nil, err
	}
	toAdd, toDelete, changed := current.metrics.diff(md)
	if len(changed) != 0 {
		return nil, errInvalidConfiguration.CausedByf("cannot alter metrics %v", changed)
	}

	return &Configuration{
		metrics:  md,
		toAdd:    toAdd,
		toDelete: toDelete,
		period:   time.Minute * time.Duration(cfg.GetGatheringPeriodMinutes()),
	}, nil
}

// Reconfigure assumes the configuration has been validated, so doesn't return
// an error.
func (tracker *TrackerBase[Finding]) Reconfigure(cfg *Configuration) {
	if cfg == nil {
		cfg = &Configuration{}
	}
	previous := tracker.setConfiguration(cfg)
	if previous != nil {
		if cfg.period == 0 {
			log.Debugf("Metrics collection has been disabled for %s", tracker.description)
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
	tracker.gatherers.Range(func(userID, g any) bool {
		for _, metric := range metrics {
			g.(*gatherer).registry.UnregisterMetric(string(metric))
		}
		return true
	})
}

func (tracker *TrackerBase[Finding]) registerMetrics(cfg *Configuration, metrics []MetricName) {
	tracker.gatherers.Range(func(userID, g any) bool {
		for _, metric := range metrics {
			tracker.registerMetric(g.(*gatherer), cfg, metric)
		}
		return true
	})
}

func (tracker *TrackerBase[Finding]) registerMetric(gatherer *gatherer, cfg *Configuration, metric MetricName) {
	if err := gatherer.registry.RegisterMetric(
		string(metric),
		tracker.description,
		cfg.period,
		labelsAsStrings(cfg.metrics[metric]),
	); err != nil {
		log.Errorf("Failed to register %s metric %q: %v", tracker.description, metric, err)
		return
	}
	log.Debugf("Registered %s Prometheus metric %q", tracker.description, metric)
}

func (tracker *TrackerBase[Finding]) getConfiguration() *Configuration {
	tracker.metricsConfigMux.RLock()
	defer tracker.metricsConfigMux.RUnlock()
	return tracker.config
}

func (tracker *TrackerBase[Finding]) setConfiguration(config *Configuration) *Configuration {
	tracker.metricsConfigMux.Lock()
	defer tracker.metricsConfigMux.Unlock()
	previous := tracker.config
	tracker.config = config
	return previous
}

// track aggregates the fetched findings and updates the gauges.
func (tracker *TrackerBase[Finding]) track(ctx context.Context, registry metrics.CustomRegistry, metrics MetricDescriptors) error {
	if len(metrics) == 0 {
		return nil
	}
	aggregator := makeAggregator(metrics, tracker.labelOrder, tracker.getters)
	for finding := range tracker.generator(ctx, metrics) {
		if err := finding.GetError(); err != nil {
			return err
		}
		aggregator.count(finding)
	}
	registry.Lock()
	defer registry.Unlock()
	for metric, records := range aggregator.result {
		registry.Reset(string(metric))
		for _, rec := range records {
			registry.SetTotal(string(metric), rec.labels, rec.total)
		}
	}
	return nil
}

// Gather the data not more often then maxAge.
func (tracker *TrackerBase[Finding]) Gather(ctx context.Context) {
	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return
	}
	cfg := tracker.getConfiguration()
	if cfg == nil {
		return
	}
	// Pass the cfg so that the same configuration is used there and here.
	gatherer := tracker.getGatherer(id.UID(), cfg)
	if gatherer == nil {
		return
	}
	defer gatherer.running.Store(false)

	if cfg.period == 0 || time.Since(gatherer.lastGather) < cfg.period {
		return
	}
	if err := tracker.track(ctx, gatherer.registry, cfg.metrics); err != nil {
		log.Errorf("Failed to gather %s metrics: %v", tracker.description, err)
	}
	gatherer.lastGather = time.Now()
}

// getGatherer returns the existing or a new gatherer for the given userID.
// The returned gatherer will be set to a running state for synchronization
// purposes. When creating a new gatherer, it also registers all known metrics
// on the gatherer registry.
// Returns nil on error, or if the gatherer for this userID is still running.
func (tracker *TrackerBase[Finding]) getGatherer(userID string, cfg *Configuration) *gatherer {
	defer tracker.cleanupInactiveGatherers()
	var gr *gatherer
	if g, ok := tracker.gatherers.Load(userID); !ok {
		r := tracker.registryFactory(userID)
		gr = &gatherer{
			registry: r,
		}
		gr.running.Store(true)
		tracker.gatherers.Store(userID, gr)
		for metricName := range cfg.metrics {
			tracker.registerMetric(gr, cfg, metricName)
		}
	} else {
		gr = g.(*gatherer)
		// Return nil if this gatherer is still running.
		// Otherwise mark it running.
		if !gr.running.CompareAndSwap(false, true) {
			return nil
		}
	}
	return gr
}

// cleanupInactiveGatherers frees the registries for the userIDs, that haven't
// shown up for inactiveGathererTTL.
func (tracker *TrackerBase[Finding]) cleanupInactiveGatherers() {
	tracker.cleanupWG.Add(1)
	go func() {
		defer tracker.cleanupWG.Done()
		tracker.gatherers.Range(func(userID, g any) bool {
			if g, ok := g.(*gatherer); ok && !g.running.Load() &&
				time.Since(g.lastGather) >= inactiveGathererTTL &&
				// Do not delete a just created gatherer in test.
				// Not in test the lastGather should never be zero for a
				// non-running gatherer.
				!g.lastGather.IsZero() {
				metrics.DeleteCustomRegistry(userID.(string))
				tracker.gatherers.Delete(userID)
			}
			return true
		})
	}()
}
