package processsignal

import (
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/metrics"
)

const (
	interval    = 5 * time.Second
	maxLRUCache = 100000
)

type enricher struct {
	lru             *lru.Cache
	clusterEntities *clusterentities.Store
	indicators      chan *v1.ProcessIndicator
}

func newEnricher(clusterEntities *clusterentities.Store, indicators chan *v1.ProcessIndicator) *enricher {
	evictfunc := func(key interface{}, value interface{}) {
		metrics.IncrementProcessEnrichmentDrops()
	}
	lru, err := lru.NewWithEvict(maxLRUCache, evictfunc)
	if err != nil {
		logger.Error(err)
		return nil
	}
	e := &enricher{
		lru:             lru,
		clusterEntities: clusterEntities,
		indicators:      indicators,
	}
	go e.processLoop()
	return e
}

func (e *enricher) Add(indicator *v1.ProcessIndicator) {
	e.lru.Add(indicator, indicator)
	metrics.SetProcessEnrichmentCacheSize(float64(e.lru.Len()))
}

func (e *enricher) processLoop() {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		for key := range e.lru.Keys() {
			if indicator, ok := e.lru.Get(key); ok {
				e.enrich(indicator.(*v1.ProcessIndicator))
			}
		}
	}
}

func (e *enricher) enrich(indicator *v1.ProcessIndicator) {
	metadata, ok := e.clusterEntities.LookupByContainerID(indicator.GetSignal().GetContainerId())
	if ok {
		populateIndicatorFromCachedContainer(indicator, metadata)
		e.indicators <- indicator
		e.lru.Remove(indicator)
		metrics.IncrementProcessEnrichmentHits()
		metrics.SetProcessEnrichmentCacheSize(float64(e.lru.Len()))
	}
}
