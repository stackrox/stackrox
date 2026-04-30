package datastore

import (
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

	processUpsertedArgsSizeHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_upserted_args_size",
		Help:      "Distribution of process argument sizes in bytes for upserted indicators",
		Buckets:   []float64{0, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
	})

	processUpsertedArgsSizeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_upserted_args_size_total",
		Help:      "Total upserted process argument sizes in bytes by cluster and namespace",
	}, []string{"cluster", "namespace"})

	processUpsertedCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_upserted_count",
		Help:      "Number of process indicators upserted by cluster and namespace",
	}, []string{"cluster", "namespace"})

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

	processUpsertedLineageSizeHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_upserted_lineage_size",
		Help:      "Distribution of process lineage sizes in bytes for upserted indicators",
		Buckets:   []float64{0, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
	})

	processUpsertedLineageSizeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_upserted_lineage_size_total",
		Help:      "Total upserted process lineage sizes in bytes by cluster and namespace",
	}, []string{"cluster", "namespace"})
)

func recordProcessIndicatorsRemoved(num int, reason string) {
	processIndicatorsRemoved.WithLabelValues(reason).Add(float64(num))
	processIndicatorsRemovedTotal.Add(float64(num))
}

func incrementProcessPruningCacheHitsMetrics() {
	processPruningCacheHits.Inc()
}

func incrementProcessPruningCacheMissesMetric() {
	processPruningCacheMisses.Inc()
}

func getProcessArgsSizeBytes(indicator *storage.ProcessIndicator) int {
	if indicator == nil || indicator.GetSignal() == nil {
		return 0
	}
	return len(indicator.GetSignal().GetArgs())
}

func getProcessLineageSizeBytes(indicator *storage.ProcessIndicator) int {
	if indicator == nil || indicator.GetSignal() == nil {
		return 0
	}

	lineageInfo := indicator.GetSignal().GetLineageInfo()
	if len(lineageInfo) == 0 {
		return 0
	}

	totalBytes := 0
	for _, info := range lineageInfo {
		if info != nil {
			totalBytes += len(info.GetParentExecFilePath())
		}
	}

	return totalBytes
}

// recordProcessIndicatorsBatchAdded records metrics for a batch of process indicators successfully written to DB.
func recordProcessIndicatorsBatchAdded(indicators []*storage.ProcessIndicator) {
	for _, indicator := range indicators {
		argsSizeBytes := getProcessArgsSizeBytes(indicator)
		lineageSizeBytes := getProcessLineageSizeBytes(indicator)
		clusterID := indicator.GetClusterId()
		namespace := indicator.GetNamespace()

		processUpsertedArgsSizeHistogram.Observe(float64(argsSizeBytes))
		processUpsertedArgsSizeTotal.WithLabelValues(clusterID, namespace).Add(float64(argsSizeBytes))
		processUpsertedCount.WithLabelValues(clusterID, namespace).Inc()
		processUpsertedLineageSizeHistogram.Observe(float64(lineageSizeBytes))
		processUpsertedLineageSizeTotal.WithLabelValues(clusterID, namespace).Add(float64(lineageSizeBytes))
	}
}

func init() {
	prometheus.MustRegister(
		processPruningCacheHits,
		processPruningCacheMisses,
		processUpsertedArgsSizeHistogram,
		processUpsertedArgsSizeTotal,
		processUpsertedCount,
		processIndicatorsRemoved,
		processIndicatorsRemovedTotal,
		processUpsertedLineageSizeHistogram,
		processUpsertedLineageSizeTotal,
	)
}
