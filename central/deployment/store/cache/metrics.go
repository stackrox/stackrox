package cache

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		deploymentStoreCacheObjects,
		deploymentStoreCacheSize,
		deploymentStoreCacheHits,
		deploymentStoreCacheMisses,
	)
}

var (
	// Note that this metric includes tombstones
	deploymentStoreCacheObjects = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "deployment_store_cache_objects",
		Help:      "Number of objects in the deployment store cache",
	})

	deploymentStoreCacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "deployment_store_cache_size",
		Help:      "Number of bytes in the deployment store cache",
	})

	deploymentStoreCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "deployment_store_cache_hits",
		Help:      "Number of cache hits in the deployment store cache",
	})

	deploymentStoreCacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "deployment_store_cache_misses",
		Help:      "Number of cache misses in the deployment store cache",
	})
)
