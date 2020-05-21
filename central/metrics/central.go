package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
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

	badgerOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "badger_op_duration",
		Help:      "Time taken to perform a badger operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	rocksDBOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "rocksdb_op_duration",
		Help:      "Time taken to perform a rocksdb operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	dackboxOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "dackbox_op_duration",
		Help:      "Time taken to perform a dackbox operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	graphQLOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "graphql_op_duration",
		Help:      "Time taken to run a single graphql sub resolver/sub query",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Resolver", "Operation"})

	graphQLQueryHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "graphql_query_duration",
		Help:      "Time taken to run a single graphql query",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Query"})

	sensorEventDurationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "sensor_event_duration",
		Help:      "Time taken to perform an process a sensor event operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Type", "Action"})

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

	riskProcessingHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "risk_processing_duration",
		Help:      "Histogram of how long risk processing takes",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Risk_Reprocessor"})

	totalCacheOperationsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "total_db_cache_operations_counter",
		Help:      "A counter of the total number of DB cache operations performed on Central",
	}, []string{"Operation", "Type"})

	datastoreFunctionDurationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "datastore_function_duration",
		Help:      "Histogram of how long a datastore function takes",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Type", "Function"})

	functionSegmentDurationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "function_segment_duration",
		Help:      "Histogram of how long a particular segment within a function takes",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Segment"})
)

func startTimeToMS(t time.Time) float64 {
	return float64(time.Since(t).Nanoseconds()) / float64(time.Millisecond)
}

// SetBoltOperationDurationTime times how long a particular bolt operation took on a particular resource
func SetBoltOperationDurationTime(start time.Time, op metrics.Op, t string) {
	boltOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Observe(startTimeToMS(start))
}

// SetBadgerOperationDurationTime times how long a particular badger operation took on a particular resource
func SetBadgerOperationDurationTime(start time.Time, op metrics.Op, t string) {
	badgerOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Observe(startTimeToMS(start))
}

// SetRocksDBOperationDurationTime times how long a particular rocksdb operation took on a particular resource
func SetRocksDBOperationDurationTime(start time.Time, op metrics.Op, t string) {
	rocksDBOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Observe(startTimeToMS(start))
}

// SetDackboxOperationDurationTime times how long a particular dackbox operation took on a particular resource
func SetDackboxOperationDurationTime(start time.Time, op metrics.Op, t string) {
	dackboxOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Observe(startTimeToMS(start))
}

// SetGraphQLOperationDurationTime times how long a particular graphql API took on a particular resource
func SetGraphQLOperationDurationTime(start time.Time, resolver metrics.Resolver, op string) {
	graphQLOperationHistogramVec.With(prometheus.Labels{"Resolver": resolver.String(), "Operation": op}).Observe(startTimeToMS(start))
}

// SetGraphQLQueryDurationTime times how long a particular graphql API took on a particular resource
func SetGraphQLQueryDurationTime(start time.Time, query string) {
	graphQLQueryHistogramVec.With(prometheus.Labels{"Query": query}).Observe(startTimeToMS(start))
}

// SetSensorEventRunDuration times how long a particular sensor event operation took on a particular resource
func SetSensorEventRunDuration(start time.Time, t, action string) {
	sensorEventDurationHistogramVec.With(prometheus.Labels{"Type": t, "Action": action}).Observe(startTimeToMS(start))
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
func ObserveRiskProcessingDuration(startTime time.Time, riskObjectType string) {
	riskProcessingHistogramVec.With(prometheus.Labels{"Risk_Reprocessor": riskObjectType}).Observe(startTimeToMS(startTime))
}

// IncrementDBCacheCounter is a counter for how many times a DB cache hits and misses
func IncrementDBCacheCounter(op string, t string) {
	totalCacheOperationsCounter.With(prometheus.Labels{"Operation": op, "Type": t}).Inc()
}

// SetDatastoreFunctionDuration is a histogram for datastore function timing
func SetDatastoreFunctionDuration(start time.Time, resourceType, function string) {
	datastoreFunctionDurationHistogramVec.With(prometheus.Labels{"Type": resourceType, "Function": function}).Observe(startTimeToMS(start))
}

// SetFunctionSegmentDuration times a specific segment within a function
func SetFunctionSegmentDuration(start time.Time, segment string) {
	functionSegmentDurationHistogramVec.With(prometheus.Labels{"Segment": segment}).Observe(startTimeToMS(start))
}
