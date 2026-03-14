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
>>>>>>> 1945cff725 (X-Smart-Squash: Squashed 5 commits:)
)

func incrementPrunedProcessesMetric(num int, reason string) {
	recordProcessIndicatorsRemoved(num, reason)
}

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

// recordProcessIndicatorsBatchAdded records metrics for a batch of process indicators successfully written to DB.
func recordProcessIndicatorsBatchAdded(indicators []*storage.ProcessIndicator) {
	for _, indicator := range indicators {
		argsSizeChars := getProcessArgsSizeChars(indicator)
		clusterID := indicator.GetClusterId()
		namespace := indicator.GetNamespace()

		processUpsertedArgsSizeHistogram.Observe(float64(argsSizeChars))

		processUpsertedArgsSizeTotal.WithLabelValues(clusterID, namespace).Add(float64(argsSizeChars))
		processUpsertedCount.WithLabelValues(clusterID, namespace).Inc()
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
	)
}
