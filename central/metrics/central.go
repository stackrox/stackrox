package metrics

import (
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
)

// IncrementPanicCounter increments the number of panic calls seen in a function
func IncrementPanicCounter(functionName string) {
	panicCounter.With(prometheus.Labels{"FunctionName": functionName}).Inc()
}
