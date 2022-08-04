package processsignal

import (
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/metrics"
)

const (
	enrichInterval = 5 * time.Second
	maxLRUCache    = 100000

	pruneInterval       = 5 * time.Minute
	containerExpiration = 30 * time.Second
)

type enricher struct {
	lru                  *lru.Cache
	clusterEntities      *clusterentities.Store
	indicators           chan *storage.ProcessIndicator
	metadataCallbackChan <-chan clusterentities.ContainerMetadata
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

// fetchAndClearProcesses returns all the processes in the given *containerWrap
// and then clears them from the *containerWrap.
// This function is atomic.
func (cw *containerWrap) fetchAndClearProcesses() []*storage.ProcessIndicator {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	processes := cw.processes
	cw.processes = nil
	return processes
}

func newEnricher(clusterEntities *clusterentities.Store, indicators chan *storage.ProcessIndicator) *enricher {
	evictfunc := func(key interface{}, value interface{}) {
		metrics.IncrementProcessEnrichmentDrops()
	}
	lru, err := lru.NewWithEvict(maxLRUCache, evictfunc)
	if err != nil {
		panic(err)
	}

	callbackChan := make(chan clusterentities.ContainerMetadata)
	if oldC := clusterEntities.RegisterContainerMetadataCallbackChannel(callbackChan); oldC != nil {
		log.Panic("Multiple container metadata callback channels registered on cluster entities store!")
	}
	e := &enricher{
		lru:                  lru,
		clusterEntities:      clusterEntities,
		indicators:           indicators,
		metadataCallbackChan: callbackChan,
	}
	go e.processLoop()
	return e
}

func (e *enricher) Add(indicator *storage.ProcessIndicator) {
	if indicator == nil {
		return
	}
	var wrap *containerWrap
	wrapObj, ok := e.lru.Get(indicator.GetSignal().GetContainerId())
	if !ok {
		wrap = &containerWrap{
			expiration: time.Now().Add(containerExpiration),
		}
	} else {
		wrap = wrapObj.(*containerWrap)
	}

	wrap.addProcess(indicator)
	e.lru.Add(indicator.GetSignal().GetContainerId(), wrap)
	metrics.SetProcessEnrichmentCacheSize(float64(e.lru.Len()))
}

func (e *enricher) processLoop() {
	ticker := time.NewTicker(enrichInterval)
	expirationTicker := time.NewTicker(pruneInterval)
	for {
		select {
		// unresolved indicators
		case <-ticker.C:
			for _, containerID := range e.lru.Keys() {
				if metadata, ok := e.clusterEntities.LookupByContainerID(containerID.(string)); ok {
					e.scanAndEnrich(metadata)
				}
			}
		case <-expirationTicker.C:
			for _, containerID := range e.lru.Keys() {
				wrapObj, exists := e.lru.Peek(containerID)
				if !exists {
					continue
				}
				wrap := wrapObj.(*containerWrap)
				// If the current value has not expired, then break because all the next values are newer
				if wrap.expiration.After(time.Now()) {
					break
				}
				e.lru.Remove(containerID)
			}
			metrics.SetProcessEnrichmentCacheSize(float64(e.lru.Len()))
		// call backs
		case metadata := <-e.metadataCallbackChan:
			e.scanAndEnrich(metadata)
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
		processes := wrapObj.(*containerWrap).fetchAndClearProcesses()
		for _, indicator := range processes {
			e.enrich(indicator, metadata)
		}
	}
}

func (e *enricher) enrich(indicator *storage.ProcessIndicator, metadata clusterentities.ContainerMetadata) {
	populateIndicatorFromCachedContainer(indicator, metadata)
	e.indicators <- indicator
	metrics.IncrementProcessEnrichmentHits()
	metrics.SetProcessEnrichmentCacheSize(float64(e.lru.Len()))
}
