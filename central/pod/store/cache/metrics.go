package cache

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/stackrox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		podStoreCacheObjects,
		podStoreCacheSize,
		podStoreCacheHits,
		podStoreCacheMisses,
	)
}

var (
	// Note that this metric includes tombstones
	podStoreCacheObjects = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "pod_store_cache_objects",
		Help:      "Number of objects in the pod store cache",
	})

	podStoreCacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "pod_store_cache_size",
		Help:      "Number of bytes in the pod store cache",
	})

	podStoreCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "pod_store_cache_hits",
		Help:      "Number of cache hits in the pod store cache",
	})

	podStoreCacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "pod_store_cache_misses",
		Help:      "Number of cache misses in the pod store cache",
	})
)
