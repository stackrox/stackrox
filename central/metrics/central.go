package metrics

import (
	"time"

	"bitbucket.org/stack-rox/apollo/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
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

	searchDurationHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.Namespace,
		Subsystem: subsystem,
		Name:      "search_duration_ms",
		Help:      "Time taken to process search query",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	})
)

// IncrementPanicCounter increments the number of panic calls seen in a function
func IncrementPanicCounter(functionName string) {
	panicCounter.With(prometheus.Labels{"FunctionName": functionName}).Inc()
}

//SetAPIRequestDurationTime records the duration of a search request
func SetAPIRequestDurationTime(took time.Duration) {
	ms := float64(took.Nanoseconds()) / float64(time.Millisecond)
	searchDurationHistogram.Observe(ms)
}
