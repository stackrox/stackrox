package datastore

import (
	"unicode/utf8"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
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

	processUpsertedArgsSizeHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_upserted_args_size",
		Help:      "Distribution of process argument sizes in characters for upserted indicators",
		Buckets:   []float64{0, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
	}, []string{"cluster"})
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

// getProcessArgsSizeChars safely calculates the size of process args in characters (runes).
// Returns 0 if signal or args are nil/empty.
func getProcessArgsSizeChars(indicator *storage.ProcessIndicator) int {
	if indicator == nil || indicator.GetSignal() == nil {
		return 0
	}
	return utf8.RuneCountInString(indicator.GetSignal().GetArgs())
}

// recordProcessIndicatorsBatchAdded records metrics for a batch of process indicators successfully written to DB.
func recordProcessIndicatorsBatchAdded(indicators []*storage.ProcessIndicator) {
	for _, indicator := range indicators {
		argsSizeChars := getProcessArgsSizeChars(indicator)
		clusterID := indicator.GetClusterId()
		processUpsertedArgsSizeHistogram.WithLabelValues(clusterID).Observe(float64(argsSizeChars))
	}
}

func init() {
	prometheus.MustRegister(
		prunedProcesses,
		processPruningCacheHits,
		processPruningCacheMisses,
		processUpsertedArgsSizeHistogram,
	)
}
