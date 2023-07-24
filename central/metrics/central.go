package metrics

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/sensor/event"
	"github.com/stackrox/rox/pkg/stringutils"
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

	postgresOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_op_duration",
		Help:      "Time taken to perform a postgres operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	acquireDBConnHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_acquire_conn_op_duration",
		Help:      "Time taken to acquire a Postgres connection",
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

	totalNetworkEndpointsReceivedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "total_network_endpoints_received_counter",
		Help:      "A counter of the total number of network endpoints received by Central from Sensor",
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

	k8sObjectProcessingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "k8s_event_processing_duration",
		Help:      "Time taken to fully process an event from Kubernetes",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Action", "Resource", "Dispatcher"})

	clusterMetricsNodeCountGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "cluster_metrics_node_count",
		Help:      "Number of nodes in a secured cluster",
	}, []string{"ClusterID"})

	clusterMetricsCPUCapacityGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "cluster_metrics_cpu_capacity",
		Help:      "Total Kubernetes cpu capacity of all nodes in a secured cluster",
	}, []string{"ClusterID"})

	totalOrphanedPLOPCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "orphaned_plop_total",
		Help:      "A counter of the total number of PLOP objects without a reference to a ProcessIndicator",
	}, []string{"ClusterID"})

	processQueueLengthGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_queue_length",
		Help:      "A gauge that indicates the current number of processes that have not been flushed",
	})

	sensorEventsDeduperCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "sensor_event_deduper",
		Help:      "A counter that tracks objects that has passed the sensor event deduper in the connection stream",
	}, []string{"status", "type"})

	pipelinePanicCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "pipeline_panics",
		Help:      "A counter that tracks the number of panics that have occurred in the processing pipelines",
	}, []string{"resource"})
	sensorConnectedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "sensor_connected",
	}, []string{"ClusterID", "reconnect"})
)

func startTimeToMS(t time.Time) float64 {
	return float64(time.Since(t).Nanoseconds()) / float64(time.Millisecond)
}

// SetPostgresOperationDurationTime times how long a particular postgres operation took on a particular resource
func SetPostgresOperationDurationTime(start time.Time, op metrics.Op, t string) {
	postgresOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).
		Observe(startTimeToMS(start))
}

// SetAcquireDBConnDuration times how long it took the database pool to acquire a connection
func SetAcquireDBConnDuration(start time.Time, op metrics.Op, t string) {
	acquireDBConnHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Observe(startTimeToMS(start))
}

// SetGraphQLOperationDurationTime times how long a particular graphql API took on a particular resource
func SetGraphQLOperationDurationTime(start time.Time, resolver metrics.Resolver, op string) {
	graphQLOperationHistogramVec.With(prometheus.Labels{"Resolver": resolver.String(), "Operation": op}).
		Observe(startTimeToMS(start))
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
	indexOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).
		Observe(startTimeToMS(start))
}

// IncrementPipelinePanics increments the counter tracking the panics in pipeline processing
func IncrementPipelinePanics(msg *central.MsgFromSensor) {
	resource := reflectutils.Type(msg.GetMsg())
	if event := msg.GetEvent(); event != nil {
		resource = reflectutils.Type(event.GetResource())
	}
	resource = stringutils.GetAfterLast(resource, "_")
	pipelinePanicCounter.With(prometheus.Labels{"resource": resource}).Inc()
}

func IncrementSensorConnect(cluserID string, reconnect bool) {
	sensorConnectedCounter.With(prometheus.Labels{"ClusterID": cluserID, "reconnect": fmt.Sprintf("%s", reconnect)}).Inc()
}

// IncrementSensorEventQueueCounter increments the counter for the passed operation
func IncrementSensorEventQueueCounter(op metrics.Op, t string) {
	sensorEventQueueCounterVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Inc()
}

// IncrementResourceProcessedCounter is a counter for how many times a resource has been processed in Central
func IncrementResourceProcessedCounter(op metrics.Op, resource metrics.Resource) {
	resourceProcessedCounterVec.With(prometheus.Labels{"Operation": op.String(), "Resource": resource.String()}).Inc()
}

// IncrementTotalNetworkFlowsReceivedCounter registers the total number of flows received
func IncrementTotalNetworkFlowsReceivedCounter(clusterID string, numberOfFlows int) {
	totalNetworkFlowsReceivedCounter.With(prometheus.Labels{"ClusterID": clusterID}).Add(float64(numberOfFlows))
}

// IncrementTotalNetworkEndpointsReceivedCounter registers the total number of endpoints received
func IncrementTotalNetworkEndpointsReceivedCounter(clusterID string, numberOfEndpoints int) {
	totalNetworkEndpointsReceivedCounter.With(prometheus.Labels{"ClusterID": clusterID}).Add(float64(numberOfEndpoints))
}

// ObserveRiskProcessingDuration adds an observation for risk processing duration.
func ObserveRiskProcessingDuration(startTime time.Time, riskObjectType string) {
	riskProcessingHistogramVec.With(prometheus.Labels{"Risk_Reprocessor": riskObjectType}).
		Observe(startTimeToMS(startTime))
}

// SetDatastoreFunctionDuration is a histogram for datastore function timing
func SetDatastoreFunctionDuration(start time.Time, resourceType, function string) {
	datastoreFunctionDurationHistogramVec.With(prometheus.Labels{"Type": resourceType, "Function": function}).
		Observe(startTimeToMS(start))
}

// SetFunctionSegmentDuration times a specific segment within a function
func SetFunctionSegmentDuration(start time.Time, segment string) {
	functionSegmentDurationHistogramVec.With(prometheus.Labels{"Segment": segment}).Observe(startTimeToMS(start))
}

// SetResourceProcessingDuration is the duration from sensor ingestion to Central processing
func SetResourceProcessingDuration(event *central.SensorEvent) {
	metrics.SetResourceProcessingDurationForEvent(k8sObjectProcessingDuration, event, "")
}

// SetClusterMetrics sets cluster metrics to the values that have been collected by Sensor.
func SetClusterMetrics(clusterID string, clusterMetrics *central.ClusterMetrics) {
	clusterMetricsNodeCountGaugeVec.With(prometheus.Labels{"ClusterID": clusterID}).
		Set(float64(clusterMetrics.GetNodeCount()))
	clusterMetricsCPUCapacityGaugeVec.With(prometheus.Labels{"ClusterID": clusterID}).
		Set(float64(clusterMetrics.GetCpuCapacity()))
}

// IncrementOrphanedPLOPCounter increments the counter for orphaned PLOP
// objects. An orphaned PLOP objects indicates that something is not quite
// right, e.g. process information is received after the endpoint, or not
// received at all. This type of situations require investigation.
func IncrementOrphanedPLOPCounter(clusterID string) {
	totalOrphanedPLOPCounter.With(prometheus.Labels{"ClusterID": clusterID}).Inc()
}

// ModifyProcessQueueLength modifies the metric for the number of processes that have not been flushed
func ModifyProcessQueueLength(delta int) {
	processQueueLengthGauge.Add(float64(delta))
}

// IncSensorEventsDeduper increments the sensor events deduper on whether or not it was deduped or not
func IncSensorEventsDeduper(deduped bool, msg *central.MsgFromSensor) {
	if msg.GetEvent() == nil {
		return
	}
	label := "passed"
	if deduped {
		label = "deduped"
	}
	typ := event.GetEventTypeWithoutPrefix(msg.GetEvent().GetResource())
	sensorEventsDeduperCounter.With(prometheus.Labels{"status": label, "type": typ}).Inc()
}
