package tracker

import (
	"cmp"
	"context"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/telemetry/centralclient"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"github.com/stackrox/rox/pkg/utils"
)

const inactiveGathererTTL = 2 * 24 * time.Hour

var (
	log = logging.CreateLogger(logging.ModuleForName("central_metrics"), 1)
)

// Getter is a function, normally bound to a label, that extracts the label
// value from a finding.
type Getter[F Finding] func(F) string

// LazyLabelGetters enables deferred evaluation of a label's value.
// Computing and storing values for all labels for every finding would be
// inefficient. Instead, the Getter function computes the value for this
// specific label only when provided with a finding.
type LazyLabelGetters[F Finding] map[Label]Getter[F]

// GetLabels returns a slice of labels from the list of lazy getters.
func (ll LazyLabelGetters[F]) GetLabels() []string {
	result := make([]string, 0, len(ll))
	for _, label := range slices.Sorted(maps.Keys(ll)) {
		result = append(result, string(label))
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

// FindingErrorSequence is a sequence of pairs of findings and errors.
type FindingErrorSequence[F Finding] = iter.Seq2[F, error]

// FindingGenerator returns an iterator to the sequence of findings.
type FindingGenerator[F Finding] func(context.Context, MetricDescriptors) FindingErrorSequence[F]

type gatherer[F Finding] struct {
	http.Handler
	lastGather time.Time
	running    atomic.Bool
	registry   metrics.CustomRegistry
	aggregator *aggregator[F]
	config     *Configuration
}

// updateMetrics aggregates the fetched findings and updates the gauges.
func (g *gatherer[F]) updateMetrics(generator iter.Seq2[F, error]) error {
	g.aggregator.reset()
	for finding, err := range generator {
		if err != nil {
			return err
		}
		g.aggregator.count(finding)
	}
	g.registry.Lock()
	defer g.registry.Unlock()
	for metric, records := range g.aggregator.result {
		g.registry.Reset(string(metric))
		for _, rec := range records {
			g.registry.SetTotal(string(metric), rec.labels, rec.total)
		}
	}
	return nil
}

func (g *gatherer[F]) trySetRunning() bool {
	return g.running.CompareAndSwap(false, true)
}

// TrackerBase implements a generic finding tracker.
// Configured with a finding generator and other arguments, it runs a goroutine
// that periodically aggregates gathered values and updates the gauge values.
type TrackerBase[F Finding] struct {
	metricPrefix string
	description  string
	getters      LazyLabelGetters[F]
	generator    FindingGenerator[F]

	// metricsConfig can be changed with an API call.
	config           *Configuration
	metricsConfigMux sync.RWMutex

	gatherers sync.Map       // map[user ID]*gatherer
	cleanupWG sync.WaitGroup // for sync in testing.

	registryFactory func(userID string) (metrics.CustomRegistry, error) // for mocking in tests.
}

// MakeTrackerBase initializes a tracker without any period or metrics
// configuration. Call Reconfigure to configure the period and the metrics.
func MakeTrackerBase[F Finding](metricPrefix, description string,
	getters LazyLabelGetters[F], generator FindingGenerator[F],
) *TrackerBase[F] {
	return &TrackerBase[F]{
		metricPrefix:    metricPrefix,
		description:     description,
		getters:         getters,
		generator:       generator,
		registryFactory: metrics.GetCustomRegistry,
	}
}

// NewConfiguration does not apply the configuration.
func (tracker *TrackerBase[F]) NewConfiguration(cfg *storage.PrometheusMetrics_Group) (*Configuration, error) {
	current := tracker.getConfiguration()
	if current == nil {
		current = &Configuration{}
	}

	md, incFilters, excFilters, err := tracker.translateStorageConfiguration(cfg.GetDescriptors())
	if err != nil {
		return nil, err
	}
	toAdd, toDelete, changed := current.metrics.diff(md)
	if len(changed) != 0 {
		return nil, errInvalidConfiguration.CausedByf("cannot alter metrics %v", changed)
	}

	return &Configuration{
		metrics:        md,
		includeFilters: incFilters,
		excludeFilters: excFilters,
		toAdd:          toAdd,
		toDelete:       toDelete,
		period:         time.Minute * time.Duration(cfg.GetGatheringPeriodMinutes()),
	}, nil
}

// Reconfigure assumes the configuration has been validated, so doesn't return
// an error.
func (tracker *TrackerBase[F]) Reconfigure(cfg *Configuration) {
	if cfg == nil {
		cfg = &Configuration{}
	}
	previous := tracker.setConfiguration(cfg)
	if previous != nil {
		if !cfg.isEnabled() {
			log.Debugf("Metrics collection has been disabled for %s", tracker.description)
			tracker.unregisterMetrics(slices.Collect(maps.Keys(previous.metrics)))
			return
		}
		tracker.unregisterMetrics(cfg.toDelete)
	}
	tracker.registerMetrics(cfg, cfg.toAdd)
	// Note: aggregators are recreated lazily in getGatherer() when config
	// changes, to avoid race conditions with running gatherers.
}

func labelsAsStrings(labels []Label) []string {
	strings := make([]string, len(labels))
	for i, label := range labels {
		strings[i] = string(label)
	}
	return strings
}

func (tracker *TrackerBase[F]) unregisterMetrics(metrics []MetricName) {
	tracker.gatherers.Range(func(userID, g any) bool {
		for _, metric := range metrics {
			g.(*gatherer[F]).registry.UnregisterMetric(string(metric))
		}
		return true
	})
}

func (tracker *TrackerBase[F]) registerMetrics(cfg *Configuration, metrics []MetricName) {
	tracker.gatherers.Range(func(userID, g any) bool {
		for _, metric := range metrics {
			tracker.registerMetric(g.(*gatherer[F]), cfg, metric)
		}
		return true
	})
}

func (tracker *TrackerBase[F]) registerMetric(gatherer *gatherer[F], cfg *Configuration, metric MetricName) {
	help := formatMetricHelp(tracker.description, cfg, metric)

	if err := gatherer.registry.RegisterMetric(
		string(metric),
		help,
		labelsAsStrings(cfg.metrics[metric]),
	); err != nil {
		log.Errorf("Failed to register %s metric %q: %v", tracker.description, metric, err)
		return
	}
	log.Debugf("Registered %s Prometheus metric %q", tracker.description, metric)
}

func formatMetricHelp(description string, cfg *Configuration, metric MetricName) string {
	var help strings.Builder
	help.WriteString("The total number of ")
	help.WriteString(description)
	if len(cfg.metrics[metric]) > 0 {
		help.WriteString(" aggregated by ")
		for i, label := range cfg.metrics[metric] {
			if i > 0 {
				help.WriteString(", ")
			}
			help.WriteString(string(label))
		}
	}
	if len(cfg.includeFilters[metric]) > 0 {
		help.WriteString(", including only ")
		for i, label := range slices.Sorted(maps.Keys(cfg.includeFilters[metric])) {
			if i > 0 {
				help.WriteString(", ")
			}
			fmt.Fprintf(&help, "%s≈%q", label, cfg.includeFilters[metric][label].String())
		}
	}
	if len(cfg.excludeFilters[metric]) > 0 {
		help.WriteString(", excluding ")
		for i, label := range slices.Sorted(maps.Keys(cfg.excludeFilters[metric])) {
			if i > 0 {
				help.WriteString(", ")
			}
			fmt.Fprintf(&help, "%s≈%q", label, cfg.excludeFilters[metric][label].String())
		}
	}
	if cfg.period > 0 {
		help.WriteString(", and gathered every ")
		help.WriteString(cfg.period.String())
	}
	return help.String()
}

func (tracker *TrackerBase[F]) getConfiguration() *Configuration {
	tracker.metricsConfigMux.RLock()
	defer tracker.metricsConfigMux.RUnlock()
	return tracker.config
}

// setConfiguration updates the tracker configuration and returns the previous
// one.
func (tracker *TrackerBase[F]) setConfiguration(config *Configuration) *Configuration {
	tracker.metricsConfigMux.Lock()
	defer tracker.metricsConfigMux.Unlock()
	previous := tracker.config
	tracker.config = config
	return previous
}

// Gather the data not more often then maxAge.
func (tracker *TrackerBase[F]) Gather(ctx context.Context) {
	cfg := tracker.getConfiguration()
	if !cfg.isEnabled() {
		return
	}
	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		utils.Should(err)
		return
	}
	// Pass the cfg so that the same configuration is used there and here.
	gatherer := tracker.getGatherer(id.UID(), cfg)
	// getGatherer() returns nil if the gatherer is still running.
	if gatherer == nil {
		return
	}
	defer tracker.cleanupInactiveGatherers()
	defer gatherer.running.Store(false)

	if time.Since(gatherer.lastGather) < cfg.period {
		return
	}
	begin := time.Now()
	if err := gatherer.updateMetrics(tracker.generator(ctx, cfg.metrics)); err != nil {
		log.Errorf("Failed to gather %s metrics: %v", tracker.description, err)
	}
	end := time.Now()
	gatherer.lastGather = end

	descriptionTitle := strings.ToTitle(tracker.description[0:1]) + tracker.description[1:]
	centralclient.Singleton().Telemeter().Track(
		descriptionTitle+" metrics gathered",
		// Event property:
		map[string]any{
			descriptionTitle + " gathering seconds": uint32(end.Sub(begin).Round(time.Second).Seconds()),
		},
		// Central traits:
		telemeter.WithTraits(tracker.makeProps(descriptionTitle)),
		telemeter.WithNoDuplicates(tracker.metricPrefix))
}

func (tracker *TrackerBase[F]) makeProps(descriptionTitle string) map[string]any {
	props := make(map[string]any, 3)
	props["Total "+descriptionTitle+" metrics"] = len(tracker.config.metrics)
	props[descriptionTitle+" metrics labels"] = getLabels(tracker.config.metrics)
	return props
}

func getLabels(metrics MetricDescriptors) []Label {
	labels := set.NewSet[Label]()
	for _, metricLabels := range metrics {
		labels.AddAll(metricLabels...)
	}
	return labels.AsSortedSlice(cmp.Less)
}

// getGatherer returns the existing or a new gatherer for the given userID.
// The returned gatherer will be set to a running state for synchronization
// purposes. When creating a new gatherer, it also registers all known metrics
// on the gatherer registry and creates the aggregator.
// For existing gatherers, if the config has changed, the aggregator is recreated.
// Returns nil on error, or if the gatherer for this userID is still running.
func (tracker *TrackerBase[F]) getGatherer(userID string, cfg *Configuration) *gatherer[F] {
	var gr *gatherer[F]
	if g, ok := tracker.gatherers.Load(userID); !ok {
		r, err := tracker.registryFactory(userID)
		if err != nil {
			log.Errorw("failed to create custom registry for user", userID, logging.Err(err))
			return nil
		}
		gr = &gatherer[F]{
			registry:   r,
			aggregator: makeAggregator(cfg.metrics, cfg.includeFilters, cfg.excludeFilters, tracker.getters),
			config:     cfg,
		}
		gr.running.Store(true)
		tracker.gatherers.Store(userID, gr)
		for metricName := range cfg.metrics {
			tracker.registerMetric(gr, cfg, metricName)
		}
	} else {
		gr = g.(*gatherer[F])
		// Return nil if this gatherer is still running.
		// Otherwise mark it running.
		if !gr.trySetRunning() {
			return nil
		}
		// Recreate aggregator if config has changed since last run.
		if gr.config != cfg {
			gr.aggregator = makeAggregator(cfg.metrics, cfg.includeFilters, cfg.excludeFilters, tracker.getters)
			gr.config = cfg
		}
	}
	return gr
}

// cleanupInactiveGatherers frees the registries for the userIDs, that haven't
// shown up for inactiveGathererTTL.
func (tracker *TrackerBase[F]) cleanupInactiveGatherers() {
	tracker.cleanupWG.Add(1)
	go func() {
		defer tracker.cleanupWG.Done()
		tracker.gatherers.Range(func(userID, gv any) bool {
			g := gv.(*gatherer[F])
			// Try to make it running to not interfere with the normal gathering
			// or otherwise do nothing.
			if !g.trySetRunning() {
				return true
			}
			if time.Since(g.lastGather) >= inactiveGathererTTL &&
				// Do not delete a just created gatherer in test.
				// lastGather should never be zero for a non-running gatherer
				// in production run.
				!g.lastGather.IsZero() {
				metrics.DeleteCustomRegistry(userID.(string))
				tracker.gatherers.Delete(userID)
			} else {
				g.running.Store(false)
			}
			return true
		})
	}()
}
