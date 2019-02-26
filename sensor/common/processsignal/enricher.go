package processsignal

import (
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/metrics"
)

const (
	interval    = 5 * time.Second
	maxLRUCache = 100000
)

type enricher struct {
	lru                  *lru.Cache
	clusterEntities      *clusterentities.Store
	indicators           chan *storage.ProcessIndicator
	metadataCallbackChan <-chan clusterentities.ContainerMetadata
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
		logger.Panicf("Multiple container metadata callback channels registered on cluster entities store!")
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
	var indicatorSlice []*storage.ProcessIndicator
	if indicator == nil {
		return
	}
	indicatorObj, _ := e.lru.Get(indicator.GetSignal().GetContainerId())
	if indicatorObj != nil {
		indicatorSlice = indicatorObj.([]*storage.ProcessIndicator)
	}
	indicatorSlice = append(indicatorSlice, indicator)
	e.lru.Add(indicator.GetSignal().GetContainerId(), indicatorSlice)
	e.clusterEntities.AddCallbackForContainerMetadata(indicator.GetSignal().GetContainerId())
	metrics.SetProcessEnrichmentCacheSize(float64(e.lru.Len()))
}

func (e *enricher) processLoop() {
	ticker := time.NewTicker(interval)
	for {
		select {
		// unresolved indicators
		case <-ticker.C:
			for _, containerID := range e.lru.Keys() {
				if metadata, ok := e.clusterEntities.LookupByContainerID(containerID.(string)); ok {
					e.scanAndEnrich(metadata)
				}
			}
		// call backs
		case metadata := <-e.metadataCallbackChan:
			e.scanAndEnrich(metadata)
		}
	}
}

// scans the cache and enriches indicators that have metadata.
func (e *enricher) scanAndEnrich(metadata clusterentities.ContainerMetadata) {
	if indicatorSet, ok := e.lru.Get(metadata.ContainerID); ok {
		for _, indicator := range indicatorSet.([]*storage.ProcessIndicator) {
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
