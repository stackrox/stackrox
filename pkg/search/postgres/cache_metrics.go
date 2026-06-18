package postgres

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		cacheEntries,
		cachePopulationDuration,
		cacheBypassTotal,
		cacheMissTotal,
	)
}

var (
	cacheEntries = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "cache_entries",
		Help:      "Number of entries in each cached store",
	}, []string{"Type"})

	cachePopulationDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "cache_population_duration_seconds",
		Help:      "Time taken to populate each cached store at startup",
		Buckets:   prometheus.DefBuckets,
	}, []string{"Type"})

	cacheBypassTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "cache_bypass_total",
		Help:      "Number of queries that bypassed the cache and hit the database directly",
	}, []string{"Type", "Operation"})

	cacheMissTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "cache_miss_total",
		Help:      "Number of cache lookups where the ID was not found",
	}, []string{"Type", "Operation"})
)
