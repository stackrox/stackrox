package processsignal

import (
	"context"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/process/normalize"
	"github.com/stackrox/rox/pkg/sensor/queue"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

const (
	enrichIntervalMax = 2 * time.Minute

	pruneInterval       = 5 * time.Minute
	containerExpiration = 30 * time.Second
)

// maxLRUCacheSize returns the process enrichment cache size, scaled based
// on available memory. Default 100K is designed for enterprise clusters;
// edge clusters with tight memory get proportionally smaller caches.
func maxLRUCacheSize() int {
	size, err := queue.ScaleSize(100000)
	if err != nil || size < 100 {
		return 100 // minimum for any environment
	}
	return size
}

type enricher struct {
	lru                  *lru.Cache[string, *containerWrap]
	clusterEntities      *clusterentities.Store
	indicators           chan *storage.ProcessIndicator
	metadataCallbackChan <-chan clusterentities.ContainerMetadata
	pubSubDispatcher     common.PubSubDispatcher
	stopper              concurrency.Stopper
}

type containerWrap struct {
	mutex      sync.Mutex
	processes  []*storage.ProcessIndicator
	expiration time.Time
}

// addProcess atomically adds the given process indicator to the *containerWrap's processes.
func (cw *containerWrap) addProcess(indicator *storage.ProcessIndicator) {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	cw.processes = append(cw.processes, indicator)
}

// fetchAndClearProcesses atomically returns all the processes in the given *containerWrap
// and clears them from the *containerWrap.
func (cw *containerWrap) fetchAndClearProcesses() []*storage.ProcessIndicator {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	processes := cw.processes
	cw.processes = nil
	return processes
}

func newEnricher(ctx context.Context, clusterEntities *clusterentities.Store, pubSubDispatcher common.PubSubDispatcher) (*enricher, error) {
	evictfunc := func(key string, value *containerWrap) {
		metrics.IncrementProcessEnrichmentDrops()
	}
	unenrichedCache, err := lru.NewWithEvict[string, *containerWrap](maxLRUCacheSize(), evictfunc)
	if err != nil {
		panic(err)
	}

	callbackChan := make(chan clusterentities.ContainerMetadata)
	if oldC := clusterEntities.RegisterContainerMetadataCallbackChannel(callbackChan); oldC != nil {
		log.Panic("Multiple container metadata callback channels registered on cluster entities store!")
	}

	e := &enricher{
		lru:                  unenrichedCache,
		clusterEntities:      clusterEntities,
		indicators:           make(chan *storage.ProcessIndicator),
		metadataCallbackChan: callbackChan,
		pubSubDispatcher:     pubSubDispatcher,
		stopper:              concurrency.NewStopper(),
	}

	if features.SensorInternalPubSub.Enabled() && pubSubDispatcher != nil {
		if err := pubSubDispatcher.RegisterConsumerToLane(
			pubsub.UnenrichedProcessConsumer,
			pubsub.UnenrichedProcessIndicatorTopic,
			pubsub.UnenrichedProcessIndicatorLane,
			e.processUnenrichedIndicator,
		); err != nil {
			return nil, errors.Wrap(err, "failed to register unenriched indicators consumer")
		}
	}

	go e.processLoop(ctx)
	return e, nil
}

func (e *enricher) getEnrichedC() <-chan *storage.ProcessIndicator {
	return e.indicators
}

func (e *enricher) processUnenrichedIndicator(event pubsub.Event) error {
	unenrichedEvent, ok := event.(*UnenrichedProcessIndicatorEvent)
	if !ok {
		return errors.Errorf("unexpected event type: %T", event)
	}

	e.add(unenrichedEvent.Indicator)
	return nil
}

// add attempts to enrich the indicator immediately if container metadata is
// available, otherwise caches it for later enrichment.
//
// Thread-safety: the hashicorp/golang-lru cache is internally synchronised, so
// concurrent calls from processLoop (via scanAndEnrich) and pub/sub consumer
// callbacks are safe.
//
// Deadlock-safety: when called from an UnenrichedProcessIndicatorLane callback,
// enrich() may publish to the EnrichedProcessIndicatorLane. This is safe because
// each lane runs its own goroutine and the DefaultConsumer executes callbacks in
// a separate goroutine, so the unenriched lane is never blocked on its own
// channel. Back-pressure is possible if the enriched lane's channel is full, but
// this is bounded and will resolve as the enriched lane drains.
func (e *enricher) add(indicator *storage.ProcessIndicator) {
	if indicator == nil || indicator.GetSignal() == nil {
		return
	}

	signal := indicator.GetSignal()

	metadata, ok, _ := e.clusterEntities.LookupByContainerID(signal.GetContainerId())
	if ok {
		e.enrich(indicator, metadata)
		return
	}

	var wrap *containerWrap
	if wrapObj, ok := e.lru.Get(signal.GetContainerId()); !ok {
		wrap = &containerWrap{
			expiration: time.Now().Add(containerExpiration),
		}
	} else {
		wrap = wrapObj
	}

	wrap.addProcess(indicator)
	e.lru.Add(signal.GetContainerId(), wrap)
	metrics.SetProcessEnrichmentCacheSize(float64(e.lru.Len()))
}

func (e *enricher) Stopped() concurrency.ReadOnlyErrorSignal {
	return e.stopper.Client().Stopped()
}

func (e *enricher) processLoop(ctx context.Context) {
	defer e.stopper.Flow().ReportStopped()
	defer close(e.indicators)

	// The base interval is configurable via ROX_SENSOR_PROCESS_ENRICHER_INTERVAL.
	// Default 5s for fast enrichment; set higher (e.g. "30s", "5m") on stable/edge clusters.
	baseInterval := env.ProcessEnricherInterval.DurationSetting()
	if baseInterval <= 0 {
		baseInterval = 5 * time.Second
	}

	// Adaptive ticker: backs off to enrichIntervalMax when the LRU cache
	// is empty (no unresolved containers). Resets to baseInterval on new
	// activity. This avoids hundreds of idle scans/hour on stable clusters.
	currentInterval := baseInterval
	ticker := time.NewTicker(currentInterval)
	expirationTicker := time.NewTicker(pruneInterval)
	for {
		select {
		case <-ctx.Done():
			log.Debugf("process indicator enricher stopped: %s", ctx.Err())
			return
		case <-ticker.C:
			// Skip the full LRU scan if there's nothing to resolve.
			if e.lru.Len() == 0 {
				currentInterval = min(currentInterval*2, enrichIntervalMax)
				ticker.Reset(currentInterval)
				continue
			}
			for _, containerID := range e.lru.Keys() {
				if metadata, ok, _ := e.clusterEntities.LookupByContainerID(containerID); ok {
					e.scanAndEnrich(metadata)
				}
			}
			currentInterval = baseInterval
			ticker.Reset(currentInterval)
		case <-expirationTicker.C:
			for _, containerID := range e.lru.Keys() {
				wrap, exists := e.lru.Peek(containerID)
				if !exists {
					continue
				}
				if wrap.expiration.After(time.Now()) {
					break
				}
				e.lru.Remove(containerID)
			}
			metrics.SetProcessEnrichmentCacheSize(float64(e.lru.Len()))
		case metadata := <-e.metadataCallbackChan:
			e.scanAndEnrich(metadata)
			// Reset to fast interval on new activity.
			currentInterval = baseInterval
			ticker.Reset(currentInterval)
		}
	}
}

// scans the cache and enriches indicators that have metadata.
func (e *enricher) scanAndEnrich(metadata clusterentities.ContainerMetadata) {
	if wrapObj, ok := e.lru.Peek(metadata.ContainerID); ok {
		e.lru.Remove(metadata.ContainerID)
		// Note: it is possible another goroutine has a reference to this same *containerWrap in Add.
		// However, that process will not be dropped because either (a) it was added prior to fetchAndClearProcesses()
		// or (b) it is added after fetchAndClearProcesses(). In case (b), Add will add the *containerWrap back into
		// the cache.
		processes := wrapObj.fetchAndClearProcesses()
		for _, indicator := range processes {
			e.enrich(indicator, metadata)
		}
	}
}

func (e *enricher) publishEnrichedIndicator(indicator *storage.ProcessIndicator) {
	if features.SensorInternalPubSub.Enabled() && e.pubSubDispatcher != nil {
		event := NewEnrichedProcessIndicatorEvent(context.Background(), indicator)
		if err := e.pubSubDispatcher.Publish(event); err != nil {
			log.Errorf("Failed to publish enriched process indicator from enricher for deployment %s with id %s: %v",
				indicator.GetDeploymentId(), indicator.GetId(), err)
			metrics.IncrementProcessEnrichmentDrops()
		}
	} else {
		e.indicators <- indicator
	}
}

func (e *enricher) enrich(indicator *storage.ProcessIndicator, metadata clusterentities.ContainerMetadata) {
	PopulateIndicatorFromContainer(indicator, metadata)
	normalize.Indicator(indicator)

	e.publishEnrichedIndicator(indicator)

	metrics.IncrementProcessEnrichmentHits()
	metrics.SetProcessEnrichmentCacheSize(float64(e.lru.Len()))
}
