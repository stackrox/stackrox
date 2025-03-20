package metrics

import (
	"time"

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

	storeCacheOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "cache_op_duration",
		Help:      "Time taken to perform a cache operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	pruningDurationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "prune_duration",
		Help:      "Time to perform a pruning operation",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Type"})

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
		Buckets: prometheus.ExponentialBuckets(4, 2, 13),
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

	totalPolicyAsCodeCRsReceivedGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "total_policy_as_code_crs_received_counter",
		Help:      "A counter of the total number of policy as code CRs that have been accepted by Central from Config Controller",
	})

	riskProcessingHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "risk_processing_duration",
		Help:      "Histogram of how long risk processing takes",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Risk_Reprocessor"})

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
		Help:      "Time taken in milliseconds to fully process an event from Kubernetes",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 12),
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
	}, []string{"ClusterID", "connection_state"})

	grpcLastMessageSizeReceived = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "grpc_last_message_size_received_bytes",
		Help:      "A gauge for last message size received per message type",
	}, []string{"Type"})

	grpcLastMessageSizeSent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "grpc_last_message_size_sent_bytes",
		Help:      "A gauge for last message size sent per message type",
	}, []string{"Type"})

	grpcMaxMessageSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "grpc_max_message_size_sent_bytes",
		Help:      "A gauge for maximum message size sent in the lifetime of this central",
	}, []string{"Type"})

	grpcError = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "grpc_error",
		Help:      "A counter for gRPC errors received in sensor connections",
	}, []string{"Code"})

	grpcSentSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "grpc_message_size_sent_bytes",
		Help:      "Histogram of sent message sizes from Central",
		Buckets: []float64{
			4_000_000,
			12_000_000,
			24_000_000,
			48_000_000,
			256_000_000,
		}, // Bucket sizes selected arbitrary based on current default limits for grpc message size
	}, []string{"Type"})

	deploymentEnhancementRoundTripDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "deployment_enhancement_duration_ms",
		Help:      "Total round trip duration in milliseconds for enhancing deployments",
		Buckets:   prometheus.LinearBuckets(500, 1000, 10),
	})

	// We use a gauge instead of a histogram because the reprocessing duration
	// is expected to vary significantly depending on the number of new images
	// and the state of the cache. This makes it inefficient to define sufficiently
	// fine-grained histogram buckets.
	reprocessorDurationGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "reprocessor_duration_seconds",
		Help:      "Duration of the reprocessor loop in seconds",
	})

	// We use a gauge instead of a histogram because the reprocessing duration
	// is expected to vary significantly depending on the number of new images
	// and the state of the cache. This makes it inefficient to define sufficiently
	// fine-grained histogram buckets.
	signatureVerificationReprocessorDurationGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "signature_verification_reprocessor_duration_seconds",
		Help:      "Duration of the signature verification reprocessor loop in seconds",
	})
)

func startTimeToMS(t time.Time) float64 {
	return float64(time.Since(t).Nanoseconds()) / float64(time.Millisecond)
}

// ObserveDeploymentEnhancementTime registers how long a sensor deployment request took
func ObserveDeploymentEnhancementTime(ms float64) {
	deploymentEnhancementRoundTripDuration.Observe(ms)
}

// ObserveSentSize registers central payload sent size.
func ObserveSentSize(messageType string, size float64) {
	grpcSentSize.With(prometheus.Labels{
		"Type": messageType,
	}).Observe(size)
}

// SetGRPCMaxMessageSizeGauge sets the maximum message size observed for message with type.
func SetGRPCMaxMessageSizeGauge(typ string, size float64) {
	grpcMaxMessageSize.With(prometheus.Labels{
		"Type": typ,
	}).Set(size)
}

// SetGRPCLastMessageSizeGauge sets last sent message size observed for message with type.
func SetGRPCLastMessageSizeGauge(typ string, size float64) {
	grpcLastMessageSizeSent.With(prometheus.Labels{
		"Type": typ,
	}).Set(size)
}

// SetGRPCLastMessageSizeReceived sets the last received message size observed for message with type.
func SetGRPCLastMessageSizeReceived(typ string, size float64) {
	grpcLastMessageSizeReceived.With(prometheus.Labels{
		"Type": typ,
	}).Set(size)
}

// RegisterGRPCError increments gRPC errors in the connection with Sensor observed by Central.
func RegisterGRPCError(code string) {
	grpcError.With(prometheus.Labels{
		"Code": code,
	}).Inc()
}

// SetCacheOperationDurationTime times how long a particular store cache operation took on a particular resource.
func SetCacheOperationDurationTime(start time.Time, op metrics.Op, t string) {
	storeCacheOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).
		Observe(startTimeToMS(start))
}

// SetPruningDuration times how long it takes to prune an entity
func SetPruningDuration(start time.Time, e string) {
	pruningDurationHistogramVec.With(prometheus.Labels{"Type": e}).
		Observe(startTimeToMS(start))
}

// SetPostgresOperationDurationTime times how long a particular postgres operation took on a particular resource.
func SetPostgresOperationDurationTime(start time.Time, op metrics.Op, t string) {
	postgresOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).
		Observe(startTimeToMS(start))
}

// SetAcquireDBConnDuration times how long it took the database pool to acquire a connection.
func SetAcquireDBConnDuration(start time.Time, op metrics.Op, t string) {
	acquireDBConnHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Observe(startTimeToMS(start))
}

// SetGraphQLOperationDurationTime times how long a particular graphql API took on a particular resource.
func SetGraphQLOperationDurationTime(start time.Time, resolver metrics.Resolver, op string) {
	graphQLOperationHistogramVec.With(prometheus.Labels{"Resolver": resolver.String(), "Operation": op}).
		Observe(startTimeToMS(start))
}

// SetGraphQLQueryDurationTime times how long a particular graphql API took on a particular resource.
func SetGraphQLQueryDurationTime(start time.Time, query string) {
	graphQLQueryHistogramVec.With(prometheus.Labels{"Query": query}).Observe(startTimeToMS(start))
}

// SetSensorEventRunDuration times how long a particular sensor event operation took on a particular resource.
func SetSensorEventRunDuration(start time.Time, t, action string) {
	sensorEventDurationHistogramVec.With(prometheus.Labels{"Type": t, "Action": action}).Observe(startTimeToMS(start))
}

// SetIndexOperationDurationTime times how long a particular index operation took on a particular resource.
func SetIndexOperationDurationTime(start time.Time, op metrics.Op, t string) {
	indexOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).
		Observe(startTimeToMS(start))
}

// IncrementPipelinePanics increments the counter tracking the panics in pipeline processing.
func IncrementPipelinePanics(msg *central.MsgFromSensor) {
	resource := reflectutils.Type(msg.GetMsg())
	if event := msg.GetEvent(); event != nil {
		resource = reflectutils.Type(event.GetResource())
	}
	resource = stringutils.GetAfterLast(resource, "_")
	pipelinePanicCounter.With(prometheus.Labels{"resource": resource}).Inc()
}

// IncrementSensorConnect increments the counter for times that a new Sensor connection was observed.
func IncrementSensorConnect(clusterID, state string) {
	sensorConnectedCounter.With(prometheus.Labels{
		"ClusterID":        clusterID,
		"connection_state": state,
	}).Inc()
}

// IncrementSensorEventQueueCounter increments the counter for the passed operation.
func IncrementSensorEventQueueCounter(op metrics.Op, t string) {
	sensorEventQueueCounterVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Inc()
}

// IncrementResourceProcessedCounter is a counter for how many times a resource has been processed in Central.
func IncrementResourceProcessedCounter(op metrics.Op, resource metrics.Resource) {
	resourceProcessedCounterVec.With(prometheus.Labels{"Operation": op.String(), "Resource": resource.String()}).Inc()
}

// IncrementTotalNetworkFlowsReceivedCounter registers the total number of flows received.
func IncrementTotalNetworkFlowsReceivedCounter(clusterID string, numberOfFlows int) {
	totalNetworkFlowsReceivedCounter.With(prometheus.Labels{"ClusterID": clusterID}).Add(float64(numberOfFlows))
}

// IncrementTotalNetworkEndpointsReceivedCounter registers the total number of endpoints received.
func IncrementTotalNetworkEndpointsReceivedCounter(clusterID string, numberOfEndpoints int) {
	totalNetworkEndpointsReceivedCounter.With(prometheus.Labels{"ClusterID": clusterID}).Add(float64(numberOfEndpoints))
}

func IncrementPolicyAsCodeCRsReceivedGauge() {
	totalPolicyAsCodeCRsReceivedGauge.Inc()
}

func DecrementPolicyAsCodeCRsReceivedGauge() {
	totalPolicyAsCodeCRsReceivedGauge.Dec()
}

// ObserveRiskProcessingDuration adds an observation for risk processing duration.
func ObserveRiskProcessingDuration(startTime time.Time, riskObjectType string) {
	riskProcessingHistogramVec.With(prometheus.Labels{"Risk_Reprocessor": riskObjectType}).
		Observe(startTimeToMS(startTime))
}

// SetDatastoreFunctionDuration is a histogram for datastore function timing.
func SetDatastoreFunctionDuration(start time.Time, resourceType, function string) {
	datastoreFunctionDurationHistogramVec.With(prometheus.Labels{"Type": resourceType, "Function": function}).
		Observe(startTimeToMS(start))
}

// SetFunctionSegmentDuration times a specific segment within a function.
func SetFunctionSegmentDuration(start time.Time, segment string) {
	functionSegmentDurationHistogramVec.With(prometheus.Labels{"Segment": segment}).Observe(startTimeToMS(start))
}

// SetResourceProcessingDuration is the duration from sensor ingestion to Central processing.
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

// ModifyProcessQueueLength modifies the metric for the number of processes that have not been flushed.
func ModifyProcessQueueLength(delta int) {
	processQueueLengthGauge.Add(float64(delta))
}

// IncSensorEventsDeduper increments the sensor events deduper on whether or not it was deduped or not.
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

// SetReprocessorDuration registers how long a reprocessing step took.
func SetReprocessorDuration(start time.Time) {
	reprocessorDurationGauge.Set(time.Since(start).Seconds())
}

// SetSignatureVerificationReprocessorDuration registers how long a signature verification reprocessing step took.
func SetSignatureVerificationReprocessorDuration(start time.Time) {
	signatureVerificationReprocessorDurationGauge.Set(time.Since(start).Seconds())
}
