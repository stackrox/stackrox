package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/globaldb/ops"
	"github.com/stackrox/rox/pkg/metrics"
)

const (
	subsystem = "central"
)

var (
	// Panics encountered
	panicCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Subsystem: subsystem,
		Name:      "panic_counter",
		Help:      "Number of panic calls within Central.",
	}, []string{"FunctionName"})

	indexOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.Namespace,
		Subsystem: subsystem,
		Name:      "index_op_duration",
		Help:      "Time taken to perform an index operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	boltOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.Namespace,
		Subsystem: subsystem,
		Name:      "bolt_op_duration",
		Help:      "Time taken to perform a bolt operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	metadataCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Subsystem: subsystem,
		Name:      "metadata_cache_hits",
		Help:      "Number of cache hits in the metadata cache",
	})
	metadataCacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Subsystem: subsystem,
		Name:      "metadata_cache_misses",
		Help:      "Number of cache misses in the metadata cache",
	})
	scanCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Subsystem: subsystem,
		Name:      "scan_cache_hits",
		Help:      "Number of cache hits in the scan cache",
	})
	scanCacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Subsystem: subsystem,
		Name:      "scan_cache_misses",
		Help:      "Number of cache misses in the scan cache",
	})
)

// IncrementPanicCounter increments the number of panic calls seen in a function
func IncrementPanicCounter(functionName string) {
	panicCounter.With(prometheus.Labels{"FunctionName": functionName}).Inc()
}

func startTimeToMS(t time.Time) float64 {
	return float64(time.Since(t).Nanoseconds()) / float64(time.Millisecond)
}

// SetBoltOperationDurationTime times how long a particular bolt operation took on a particular resource
func SetBoltOperationDurationTime(start time.Time, op ops.Op, t string) {
	boltOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Observe(startTimeToMS(start))
}

// SetIndexOperationDurationTime times how long a particular index operation took on a particular resource
func SetIndexOperationDurationTime(start time.Time, op string, t string) {
	indexOperationHistogramVec.With(prometheus.Labels{"Operation": op, "Type": t}).Observe(startTimeToMS(start))
}

// IncrementMetadataCacheHit increments the number of metadata cache hits
func IncrementMetadataCacheHit() {
	metadataCacheHits.Inc()
}

// IncrementMetadataCacheMiss increments the number of metadata cache misses
func IncrementMetadataCacheMiss() {
	metadataCacheMisses.Inc()
}

// IncrementScanCacheHit increments the number of scan cache hits
func IncrementScanCacheHit() {
	scanCacheHits.Inc()
}

// IncrementScanCacheMiss increments the number of scan cache misses
func IncrementScanCacheMiss() {
	scanCacheMisses.Inc()
}
