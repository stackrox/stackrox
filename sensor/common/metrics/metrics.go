package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.Register(processDedupeCacheHits)
	prometheus.Register(processDedupeCacheMisses)
}

var (
	// Panics encountered
	panicCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "panic_counter",
		Help:      "Number of panic calls within Sensor.",
	}, []string{"FunctionName"})

	processDedupeCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "dedupe_cache_hits",
		Help:      "A counter of the total number of times we've deduped the process passed in",
	})

	processDedupeCacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "dedupe_cache_misses",
		Help:      "A counter of the total number of times we've passed through the dedupe cache",
	})
)

// IncrementPanicCounter increments the number of panic calls seen in a function
func IncrementPanicCounter(functionName string) {
	panicCounter.With(prometheus.Labels{"FunctionName": functionName}).Inc()
}

// IncrementProcessDedupeCacheHits increments the number of times we deduped a process
func IncrementProcessDedupeCacheHits() {
	processDedupeCacheHits.Inc()
}

// IncrementProcessDedupeCacheMisses increments the number of times we failed to dedupe a process
func IncrementProcessDedupeCacheMisses() {
	processDedupeCacheMisses.Inc()
}
