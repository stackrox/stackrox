package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	// Panics encountered
	panicCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "panic_counter",
		Help:      "Number of panic calls within Central.",
	}, []string{"FunctionName"})

	indexOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.Namespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "index_op_duration",
		Help:      "Time taken to perform an index operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	boltOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.Namespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "bolt_op_duration",
		Help:      "Time taken to perform a bolt operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	sensorEventQueueCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "sensor_event_queue",
		Help:      "Number of elements in removed from the queue",
	}, []string{"Operation"})
)

// IncrementPanicCounter increments the number of panic calls seen in a function
func IncrementPanicCounter(functionName string) {
	panicCounter.With(prometheus.Labels{"FunctionName": functionName}).Inc()
}

func startTimeToMS(t time.Time) float64 {
	return float64(time.Since(t).Nanoseconds()) / float64(time.Millisecond)
}

// SetBoltOperationDurationTime times how long a particular bolt operation took on a particular resource
func SetBoltOperationDurationTime(start time.Time, op metrics.Op, t string) {
	boltOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Observe(startTimeToMS(start))
}

// SetIndexOperationDurationTime times how long a particular index operation took on a particular resource
func SetIndexOperationDurationTime(start time.Time, op metrics.Op, t string) {
	indexOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Observe(startTimeToMS(start))
}

// IncrementSensorEventQueueCounter increments the counter for the passed operation
func IncrementSensorEventQueueCounter(action string) {
	sensorEventQueueCounterVec.With(prometheus.Labels{"Operation": action}).Inc()
}
