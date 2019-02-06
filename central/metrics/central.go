package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	// Panics encountered
	panicCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "panic_counter",
		Help:      "Number of panic calls within Central.",
	}, []string{"FunctionName"})

	indexOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "index_op_duration",
		Help:      "Time taken to perform an index operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	boltOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "bolt_op_duration",
		Help:      "Time taken to perform a bolt operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	sensorEventDurationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "sensor_event_duration",
		Help:      "Time taken to perform an process a sensor event operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Type"})

	sensorEventQueueCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "sensor_event_queue",
		Help:      "Number of elements in removed from the queue",
	}, []string{"Operation", "Type"})

	resourceProcessedCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "resource_processed_count",
		Help:      "Number of elements received and processed",
	}, []string{"Operation", "Resource"})

	policyEvaluationHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "policy_evaluation_duration",
		Help:      "Histogram of how long each policy has taken to evaluate",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Policy"})

	totalNetworkFlowsReceivedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "total_network_flows_central_received_counter",
		Help:      "A counter of the total number of network flows received by Central from Sensor",
	}, []string{"ClusterID"})

	riskProcessingHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "risk_processing_duration",
		Help:      "Histogram of how long risk processing takes",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
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
func SetBoltOperationDurationTime(start time.Time, op metrics.Op, t string) {
	boltOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Observe(startTimeToMS(start))
}

// SetSensorEventRunDuration times how long a particular sensor event operation took on a particular resource
func SetSensorEventRunDuration(start time.Time, t string) {
	sensorEventDurationHistogramVec.With(prometheus.Labels{"Type": t}).Observe(startTimeToMS(start))
}

// SetIndexOperationDurationTime times how long a particular index operation took on a particular resource
func SetIndexOperationDurationTime(start time.Time, op metrics.Op, t string) {
	indexOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Observe(startTimeToMS(start))
}

// IncrementSensorEventQueueCounter increments the counter for the passed operation
func IncrementSensorEventQueueCounter(op metrics.Op, t string) {
	sensorEventQueueCounterVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Inc()
}

// SetPolicyEvaluationDurationTime is the amount of time a specific policy took
func SetPolicyEvaluationDurationTime(t time.Time, name string) {
	policyEvaluationHistogram.With(prometheus.Labels{"Policy": name}).Observe(startTimeToMS(t))
}

// IncrementResourceProcessedCounter is a counter for how many times a resource has been processed in Central
func IncrementResourceProcessedCounter(op metrics.Op, resource metrics.Resource) {
	resourceProcessedCounterVec.With(prometheus.Labels{"Operation": op.String(), "Resource": resource.String()}).Inc()
}

// IncrementTotalNetworkFlowsReceivedCounter registers the total number of flows received
func IncrementTotalNetworkFlowsReceivedCounter(clusterID string, numberOfFlows int) {
	totalNetworkFlowsReceivedCounter.With(prometheus.Labels{"ClusterID": clusterID}).Add(float64(numberOfFlows))
}

// ObserveRiskProcessingDuration adds an observation for risk processing duration.
func ObserveRiskProcessingDuration(startTime time.Time) {
	riskProcessingHistogram.Observe(startTimeToMS(startTime))
}
