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

	processUpsertedArgsSizeHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_upserted_args_size",
		Help:      "Distribution of process argument sizes in characters for upserted indicators",
		Buckets:   []float64{0, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
	})

	processUpsertedArgsSizeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_upserted_args_size_total",
		Help:      "Total upserted process argument sizes in characters by cluster and namespace",
	}, []string{"cluster", "namespace"})

	processUpsertedCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_upserted_count",
		Help:      "Number of process indicators upserted by cluster and namespace",
	}, []string{"cluster", "namespace"})

	processIndicatorsLineageSizeHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_indicators_lineage_size",
		Help:      "Distribution of process lineage sizes in characters for upserted indicators",
		Buckets:   []float64{0, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
	})

	processIndicatorsLineageSizeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_indicators_lineage_size_total",
		Help:      "Total upserted process lineage sizes in characters by cluster and namespace",
	}, []string{"cluster", "namespace"})
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
	// RuneCountInString returns the number of UTF-8 characters.
	// For ASCII this is going to be equivalent to len(), but
	// it is not going to be equivalent for special characters.
	// len() is O(1), but RuneCountInString is O(n). This should
	// not be a problem because metrics are handled async and should
	// not block other processes such as database writes.
	return utf8.RuneCountInString(indicator.GetSignal().GetArgs())
}

// getProcessLineageSizeChars safely calculates the total size of process lineage in characters (runes).
// Returns 0 if signal or lineage are nil/empty.
func getProcessLineageSizeChars(indicator *storage.ProcessIndicator) int {
	if indicator == nil || indicator.GetSignal() == nil {
		return 0
	}

	lineageInfo := indicator.GetSignal().GetLineageInfo()
	if len(lineageInfo) == 0 {
		return 0
	}

	totalChars := 0
	for _, info := range lineageInfo {
		if info != nil {
			totalChars += utf8.RuneCountInString(info.GetParentExecFilePath())
		}
	}

	return totalChars
}

// recordProcessIndicatorsBatchAdded records metrics for a batch of process indicators successfully written to DB.
func recordProcessIndicatorsBatchAdded(indicators []*storage.ProcessIndicator) {
	for _, indicator := range indicators {
		argsSizeChars := getProcessArgsSizeChars(indicator)
		lineageSizeChars := getProcessLineageSizeChars(indicator)
		clusterID := indicator.GetClusterId()
		namespace := indicator.GetNamespace()

		processUpsertedArgsSizeHistogram.Observe(float64(argsSizeChars))
		processUpsertedArgsSizeTotal.WithLabelValues(clusterID, namespace).Add(float64(argsSizeChars))
		processUpsertedCount.WithLabelValues(clusterID, namespace).Inc()
		processIndicatorsLineageSizeHistogram.Observe(float64(lineageSizeChars))
		processIndicatorsLineageSizeTotal.WithLabelValues(clusterID, namespace).Add(float64(lineageSizeChars))
	}
}

func init() {
	prometheus.MustRegister(
		prunedProcesses,
		processPruningCacheHits,
		processPruningCacheMisses,
		processUpsertedArgsSizeHistogram,
		processUpsertedArgsSizeTotal,
		processUpsertedCount,
		processIndicatorsLineageSizeHistogram,
		processIndicatorsLineageSizeTotal,
	)
}
