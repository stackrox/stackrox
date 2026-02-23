package datastore

import (
	"unicode/utf8"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/metrics"
)

const (
	// PruneReasonSimilarity represents pruning based on Jaccard similarity algorithm
	PruneReasonSimilarity = "similarity"
	// PruneReasonOrphanedByDeployment represents pruning of indicators orphaned by deleted deployments
	PruneReasonOrphanedByDeployment = "orphaned_deployment"
	// PruneReasonOrphanedByPod represents pruning of indicators orphaned by deleted pods
	PruneReasonOrphanedByPod = "orphaned_pod"

	// RemovalReasonProcessFilter represents removal during process filter cleanup
	RemovalReasonProcessFilter = "process_filter"
	// RemovalReasonPodDeletion represents removal when a pod is deleted
	RemovalReasonPodDeletion = "pod_deletion"
)

var (
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

	processArgsHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_args_size_bytes",
		Help:      "Distribution of process argument sizes in bytes for indicators written to database",
		Buckets:   []float64{0, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
	})

	processArgsCharsHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_args_size_chars",
		Help:      "Distribution of process argument sizes in characters for indicators written to database",
		Buckets:   []float64{0, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
	})

	processIndicatorsRemoved = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_indicators_removed",
		Help:      "Number of process indicators removed from the database, broken down by reason",
	}, []string{"reason"})

	processIndicatorsRemovedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_indicators_removed_total",
		Help:      "Total number of process indicators removed from the database across all reasons",
	})

	processIndicatorsNet = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_indicators_net",
		Help:      "Net count of process indicators (additions minus removals)",
	})
)

func incrementPrunedProcessesMetric(num int, reason string) {
	recordProcessIndicatorsRemoved(num, reason)
}

func recordProcessIndicatorsRemoved(num int, reason string) {
	processIndicatorsRemoved.WithLabelValues(reason).Add(float64(num))
	processIndicatorsRemovedTotal.Add(float64(num))
	processIndicatorsNet.Sub(float64(num))
}

func incrementProcessPruningCacheHitsMetrics() {
	processPruningCacheHits.Inc()
}

func incrementProcessPruningCacheMissesMetric() {
	processPruningCacheMisses.Inc()
}

// getProcessArgsSizeBytes safely calculates the size of process args in bytes.
// Returns 0 if signal or args are nil/empty.
func getProcessArgsSizeBytes(indicator *storage.ProcessIndicator) int {
	if indicator == nil || indicator.GetSignal() == nil {
		return 0
	}
	return len(indicator.GetSignal().GetArgs())
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
		argsSizeBytes := getProcessArgsSizeBytes(indicator)
		argsSizeChars := getProcessArgsSizeChars(indicator)
		processArgsHistogram.Observe(float64(argsSizeBytes))
		processArgsCharsHistogram.Observe(float64(argsSizeChars))
	}
}

func init() {
	prometheus.MustRegister(
		processPruningCacheHits,
		processPruningCacheMisses,
		processArgsHistogram,
		processArgsCharsHistogram,
		processIndicatorsRemoved,
		processIndicatorsRemovedTotal,
	)
}
