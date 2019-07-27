package datastore

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	prunedProcesses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "pruned_process_indicators",
		Help:      "Number of process indicators removed by pruning",
	})

	processPruningCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_pruning_cache_hits",
		Help:      "Number of times we hit the cache when trying to prune processes",
	})

	processPruningCacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_pruning_cache_misses",
		Help:      "Number of times we miss the cache, and have to evaluate, when trying to prune processes",
	})
)

func incrementPrunedProcessesMetric(num int) {
	prunedProcesses.Add(float64(num))
}

func incrementProcessPruningCacheHitsMetrics() {
	processPruningCacheHits.Inc()
}

func incrementProcessPruningCacheMissesMetric() {
	processPruningCacheMisses.Inc()
}

func init() {
	prometheus.MustRegister(
		prunedProcesses,
		processPruningCacheHits,
		processPruningCacheMisses,
	)
}
