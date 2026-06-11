package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"

	"github.com/stackrox/rox/generated/storage"
)

func init() {
	prometheus.MustRegister(
		processFilterCounter,
		processIndicatorsNotPersisted,
		processUpsertedArgsSizeHistogram,
		processUpsertedArgsSizeTotal,
		processUpsertedCount,
		processUpsertedLineageSizeHistogram,
		processUpsertedLineageSizeTotal,
	)
}

var (
	processFilterCounter = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: pkgMetrics.PrometheusNamespace,
		Subsystem: pkgMetrics.CentralSubsystem.String(),
		Name:      "process_filter",
		Help:      "Process filter hits and misses",
	}, []string{"Type"})

	processIndicatorsNotPersisted = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: pkgMetrics.PrometheusNamespace,
		Subsystem: pkgMetrics.CentralSubsystem.String(),
		Name:      "process_indicators_not_persisted",
		Help:      "Number of process indicators filtered out and not persisted",
	})

	processUpsertedArgsSizeHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: pkgMetrics.PrometheusNamespace,
		Subsystem: pkgMetrics.CentralSubsystem.String(),
		Name:      "process_upserted_args_size",
		Help:      "Distribution of process argument sizes in bytes for upserted indicators",
		Buckets:   []float64{0, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
	})

	processUpsertedArgsSizeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: pkgMetrics.PrometheusNamespace,
		Subsystem: pkgMetrics.CentralSubsystem.String(),
		Name:      "process_upserted_args_size_total",
		Help:      "Total upserted process argument sizes in bytes by cluster and namespace",
	}, []string{"cluster", "namespace"})

	processUpsertedCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: pkgMetrics.PrometheusNamespace,
		Subsystem: pkgMetrics.CentralSubsystem.String(),
		Name:      "process_upserted_count",
		Help:      "Number of process indicators upserted by cluster and namespace",
	}, []string{"cluster", "namespace"})

	processUpsertedLineageSizeHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: pkgMetrics.PrometheusNamespace,
		Subsystem: pkgMetrics.CentralSubsystem.String(),
		Name:      "process_upserted_lineage_size",
		Help:      "Distribution of process lineage sizes in bytes for upserted indicators",
		Buckets:   []float64{0, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
	})

	processUpsertedLineageSizeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: pkgMetrics.PrometheusNamespace,
		Subsystem: pkgMetrics.CentralSubsystem.String(),
		Name:      "process_upserted_lineage_size_total",
		Help:      "Total upserted process lineage sizes in bytes by cluster and namespace",
	}, []string{"cluster", "namespace"})
)

// ProcessFilterCounterInc increments a counter for determining how effective the process filter is
func ProcessFilterCounterInc(typ string) {
	processFilterCounter.With(prometheus.Labels{"Type": typ}).Inc()
}

// ProcessIndicatorsNotPersistedInc increments the counter for process indicators filtered out and not persisted.
func ProcessIndicatorsNotPersistedInc() {
	processIndicatorsNotPersisted.Inc()
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

// RecordProcessIndicatorReceived records metrics for a single process indicator
// received by Central, regardless of whether it is persisted.
func RecordProcessIndicatorReceived(indicator *storage.ProcessIndicator) {
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
