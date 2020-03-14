package cache

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		imageStoreCacheObjects,
		imageStoreCacheSize,
		imageStoreCacheHits,
		imageStoreCacheMisses,
	)
}

var (
	// Note that this metric includes tombstones
	imageStoreCacheObjects = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_store_cache_objects",
		Help:      "Number of objects in the image store cache",
	})

	imageStoreCacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_store_cache_size",
		Help:      "Number of bytes in the image store cache",
	})

	imageStoreCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_store_cache_hits",
		Help:      "Number of cache hits in the image store cache",
	})

	imageStoreCacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_store_cache_misses",
		Help:      "Number of cache misses in the image store cache",
	})
)
