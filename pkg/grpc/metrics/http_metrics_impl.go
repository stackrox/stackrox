package metrics

import (
	"net/http"
	"sync/atomic"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/lru"
	"github.com/stackrox/rox/pkg/sync"
)

type httpMetricsImpl struct {
	allMetricsMutex sync.RWMutex
	allMetrics      map[string]*perPathHTTPMetrics
}

type perPathHTTPMetrics struct {
	normalInvocationStats      map[int]int64
	normalInvocationStatsMutex sync.RWMutex

	panics lru.Cache[string, *int64]
}

func (h *httpMetricsImpl) WrapHandler(handler http.Handler, path string) http.Handler {
	// Prevent access to the apiCalls map while we wrap a new handler
	h.allMetricsMutex.Lock()
	defer h.allMetricsMutex.Unlock()
	panicLRU, err := lru.New[string, *int64](cacheSize)
	if err != nil {
		// This should only happen if cacheSize < 0 and that should be impossible.
		log.Infof("unable to create LRU in WrapHandler for endpoint %s with size %d", path, cacheSize)
		return handler
	}
	ppm := &perPathHTTPMetrics{
		normalInvocationStats: make(map[int]int64),
		panics:                panicLRU,
	}
	h.allMetrics[path] = ppm

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panicked := true
		defer func() {
			r := recover()
			if r == nil && !panicked {
				return
			}
			panicLocation := getPanicLocation(1)
			panicCount, ok := panicLRU.Get(panicLocation)
			if ok {
				atomic.AddInt64(panicCount, 1)
				panic(r)
			}
			initialCount := int64(0)
			panicLRU.ContainsOrAdd(panicLocation, &initialCount)
			panicCount, ok = panicLRU.Get(panicLocation)
			// This panic might have been evicted from panicLRU if we're getting a lot of them
			if ok {
				atomic.AddInt64(panicCount, 1)
			}
			panic(r)
		}()

		statusTrackingWriter := httputil.NewStatusTrackingWriter(w)
		handler.ServeHTTP(statusTrackingWriter, r)

		statusCode := statusTrackingWriter.GetStatusCode()
		if statusCode == nil {
			sc := 200
			statusCode = &sc
		}
		// Prevent access to the response code map while we update the contents
		concurrency.WithLock(&ppm.normalInvocationStatsMutex, func() {
			ppm.normalInvocationStats[*statusCode]++
		})
		panicked = false
	})
}

func (h *httpMetricsImpl) GetMetrics() (map[string]map[int]int64, map[string]map[string]int64) {
	// Prevent new paths from being added to apiCalls while we iterate over it
	h.allMetricsMutex.RLock()
	defer h.allMetricsMutex.RUnlock()
	externalMetrics := make(map[string]map[int]int64, len(h.allMetrics))
	externalPanics := make(map[string]map[string]int64, len(h.allMetrics))
	for path, ppm := range h.allMetrics {
		// Prevent the response code map from being updated while we copy it
		concurrency.WithLock(&ppm.normalInvocationStatsMutex, func() {
			externalCodeMap := make(map[int]int64, len(ppm.normalInvocationStats))
			for responseCode, count := range ppm.normalInvocationStats {
				externalCodeMap[responseCode] = count
			}
			if len(externalCodeMap) > 0 {
				externalMetrics[path] = externalCodeMap
			}
		})

		// No need to lock explicitly as this LRU implementation is thread safe
		panicLocations := ppm.panics.Keys()
		panicMap := make(map[string]int64, len(panicLocations))
		for _, panicLocation := range panicLocations {
			if panicCount, ok := ppm.panics.Get(panicLocation); ok {
				panicMap[panicLocation] = atomic.LoadInt64(panicCount)
			}
		}
		if len(panicMap) > 0 {
			externalPanics[path] = panicMap
		}
	}

	return externalMetrics, externalPanics
}
