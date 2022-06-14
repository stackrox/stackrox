package processsignal

import (
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/sensor/common/clusterentities"
	"github.com/stackrox/stackrox/sensor/common/metrics"
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
	processes  []*storage.ProcessIndicator
	expiration time.Time
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

	wrap.processes = append(wrap.processes, indicator)
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
				val, exists := e.lru.Peek(containerID)
				if !exists {
					continue
				}
				wrap := val.(*containerWrap)
				// If the current value has not expired, then break because all of the next values are newer
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
	if wrapInterface, ok := e.lru.Peek(metadata.ContainerID); ok {
		wrap := wrapInterface.(*containerWrap)
		for _, indicator := range wrap.processes {
			e.enrich(indicator, metadata)
		}
		e.lru.Remove(metadata.ContainerID)
	}
}

func (e *enricher) enrich(indicator *storage.ProcessIndicator, metadata clusterentities.ContainerMetadata) {
	populateIndicatorFromCachedContainer(indicator, metadata)
	e.indicators <- indicator
	metrics.IncrementProcessEnrichmentHits()
	metrics.SetProcessEnrichmentCacheSize(float64(e.lru.Len()))
}
