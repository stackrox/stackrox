package datastore

import (
	"github.com/prometheus/client_golang/prometheus"
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

func init() {
	prometheus.MustRegister(
		processPruningCacheHits,
		processPruningCacheMisses,
		processIndicatorsRemoved,
		processIndicatorsRemovedTotal,
	)
}
