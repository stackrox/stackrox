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
	IndexOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "index_op_duration",
		Help:      "Time spent performing an index operation in milliseconds",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	StoreCacheOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "cache_op_duration",
		Help:      "Time spent performing a cache operation in milliseconds",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	PruningDurationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "prune_duration",
		Help:      "Time to perform a pruning operation in milliseconds",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Type"})

	PostgresOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_op_duration",
		Help:      "Time spent performing a postgres operation in milliseconds",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	AcquireDBConnHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_acquire_conn_op_duration",
		Help:      "Time spent acquiring a Postgres connection in milliseconds",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Operation", "Type"})

	GraphQLOperationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "graphql_op_duration",
		Help:      "Time spent running a single graphql sub resolver/sub query in milliseconds",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Resolver", "Operation"})

	GraphQLQueryHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "graphql_query_duration",
		Help:      "Time spent running a single graphql query in milliseconds",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Query"})

	SensorEventDurationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "sensor_event_duration",
		Help:      "Time spent performing a sensor event operation in milliseconds",
		// We care more about precision at lower latencies, or outliers at higher latencies.
		Buckets: prometheus.ExponentialBuckets(4, 2, 13),
	}, []string{"Type", "Action"})

	SensorEventQueueCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "sensor_event_queue",
		Help:      "Number of enqueue and dequeue operations on Central's event deduping queues for messages arriving from Sensor",
	}, []string{"Operation", "Type"})

	ResourceProcessedCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "resource_processed_count",
		Help:      "Number of sensor event resources successfully processed by Central pipelines",
	}, []string{"Operation", "Resource"})

	TotalNetworkFlowsReceivedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "total_network_flows_central_received_counter",
		Help:      "A counter of the total number of network flows received by Central from Sensor",
	}, []string{"ClusterID"})

	TotalNetworkEndpointsReceivedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "total_network_endpoints_received_counter",
		Help:      "A counter of the total number of network endpoints received by Central from Sensor",
	}, []string{"ClusterID"})

	// TotalExternalPoliciesGauge is deprecated due to naming confusion (vector vs. gauge), use CurrentExternalPolicies instead.
	TotalExternalPoliciesGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "total_external_policies_count",
		Help:      "Number of policy-as-code CRs accepted by Central from Config Controller",
	})
	// CurrentExternalPolicies replaces the TotalExternalPoliciesGauge
	CurrentExternalPolicies = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "number_of_external_policies_current",
		Help:      "Number of policy-as-code CRs accepted by Central from Config Controller",
	})

	RiskProcessingHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "risk_processing_duration",
		Help:      "Time in milliseconds spent recomputing and persisting risk scores in Central's risk manager for deployments, nodes, images, and their components",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Risk_Reprocessor"})

	DatastoreFunctionDurationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "datastore_function_duration",
		Help:      "Histogram of how long a datastore function takes in milliseconds",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Type", "Function"})

	FunctionSegmentDurationHistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "function_segment_duration",
		Help:      "Histogram of how long a particular segment within a function takes in milliseconds",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Segment"})

	K8sObjectProcessingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "k8s_event_processing_duration",
		Help:      "Time taken in milliseconds to fully process an event from Kubernetes",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 12),
	}, []string{"Action", "Resource", "Dispatcher"})

	ClusterMetricsNodeCountGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "cluster_metrics_node_count",
		Help:      "Number of nodes in a secured cluster",
	}, []string{"ClusterID"})

	ClusterMetricsCPUCapacityGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "cluster_metrics_cpu_capacity",
		Help:      "Total Kubernetes cpu capacity of all nodes in a secured cluster",
	}, []string{"ClusterID"})

	TotalOrphanedPLOPCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "orphaned_plop_total",
		Help:      "Count of process-listening-on-port records that arrived without a matching process indicator",
	}, []string{"ClusterID"})

	ProcessQueueLengthGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_queue_length",
		Help:      "Current number of process indicators queued for baseline evaluation and persistence",
	})

	SensorEventsDeduperCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "sensor_event_deduper",
		Help:      "Counts sensor events skipped by the deduper vs processed as new",
	}, []string{"status", "type"})

	PipelinePanicCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "pipeline_panics",
		Help:      "Count of panics recovered in Central processing pipelines",
	}, []string{"resource"})

	SensorConnectedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "sensor_connected",
		Help:      "Count of sensor connections observed by Central",
	}, []string{"ClusterID", "connection_state"})

	GrpcLastMessageSizeReceived = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "grpc_last_message_size_received_bytes",
		Help:      "A gauge for last message size received per message type",
	}, []string{"Type"})

	GrpcLastMessageSizeSent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "grpc_last_message_size_sent_bytes",
		Help:      "A gauge for last message size sent per message type",
	}, []string{"Type"})

	GrpcMaxMessageSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "grpc_max_message_size_sent_bytes",
		Help:      "A gauge for maximum message size sent in the lifetime of this central",
	}, []string{"Type"})

	GrpcError = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "grpc_error",
		Help:      "A counter for gRPC errors received in sensor connections",
	}, []string{"Code"})

	GrpcSentSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
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

	DeploymentEnhancementRoundTripDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
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
	ReprocessorDurationGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "reprocessor_duration_seconds",
		Help:      "Duration of the reprocessor loop in seconds",
	})

	// We use a gauge instead of a histogram because the reprocessing duration
	// is expected to vary significantly depending on the number of new images
	// and the state of the cache. This makes it inefficient to define sufficiently
	// fine-grained histogram buckets.
	SignatureVerificationReprocessorDurationGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "signature_verification_reprocessor_duration_seconds",
		Help:      "Duration of the signature verification reprocessor loop in seconds",
	})

	MsgToSensorNotSentCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "msg_to_sensor_not_sent_count",
		Help:      "Total messages not sent to Sensor due to errors or other reasons",
	}, []string{"ClusterID", "type", "reason"})
)

// Reasons for a message not being sent.
var (
	// NotSentError indicates that an attempt was made to send the message
	// but an error was encountered.
	NotSentError = "error"
	// NotSentSignal indicates that a signal prevented the message from being
	// sent, such as a timeout.
	NotSentSignal = "signal"
	// NotSentSkip indicates that no attempt was made to send the message,
	// perhaps due to prior errors.
	NotSentSkip = "skip"
)

func startTimeToMS(t time.Time) float64 {
	return float64(time.Since(t).Nanoseconds()) / float64(time.Millisecond)
}

// ObserveDeploymentEnhancementTime registers how long a sensor deployment request took
func ObserveDeploymentEnhancementTime(ms float64) {
	DeploymentEnhancementRoundTripDuration.Observe(ms)
}

// ObserveSentSize registers central payload sent size.
func ObserveSentSize(messageType string, size float64) {
	GrpcSentSize.With(prometheus.Labels{
		"Type": messageType,
	}).Observe(size)
}

// SetGRPCMaxMessageSizeGauge sets the maximum message size observed for message with type.
func SetGRPCMaxMessageSizeGauge(typ string, size float64) {
	GrpcMaxMessageSize.With(prometheus.Labels{
		"Type": typ,
	}).Set(size)
}

// SetGRPCLastMessageSizeGauge sets last sent message size observed for message with type.
func SetGRPCLastMessageSizeGauge(typ string, size float64) {
	GrpcLastMessageSizeSent.With(prometheus.Labels{
		"Type": typ,
	}).Set(size)
}

// SetGRPCLastMessageSizeReceived sets the last received message size observed for message with type.
func SetGRPCLastMessageSizeReceived(typ string, size float64) {
	GrpcLastMessageSizeReceived.With(prometheus.Labels{
		"Type": typ,
	}).Set(size)
}

// RegisterGRPCError increments gRPC errors in the connection with Sensor observed by Central.
func RegisterGRPCError(code string) {
	GrpcError.With(prometheus.Labels{
		"Code": code,
	}).Inc()
}

// SetCacheOperationDurationTime times how long a particular store cache operation took on a particular resource.
func SetCacheOperationDurationTime(start time.Time, op metrics.Op, t string) {
	StoreCacheOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).
		Observe(startTimeToMS(start))
}

// SetPruningDuration times how long it takes to prune an entity
func SetPruningDuration(start time.Time, e string) {
	PruningDurationHistogramVec.With(prometheus.Labels{"Type": e}).
		Observe(startTimeToMS(start))
}

// SetPostgresOperationDurationTime times how long a particular postgres operation took on a particular resource.
func SetPostgresOperationDurationTime(start time.Time, op metrics.Op, t string) {
	PostgresOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).
		Observe(startTimeToMS(start))
}

// SetAcquireDBConnDuration times how long it took the database pool to acquire a connection.
func SetAcquireDBConnDuration(start time.Time, op metrics.Op, t string) {
	AcquireDBConnHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Observe(startTimeToMS(start))
}

// SetGraphQLOperationDurationTime times how long a particular graphql API took on a particular resource.
func SetGraphQLOperationDurationTime(start time.Time, resolver metrics.Resolver, op string) {
	GraphQLOperationHistogramVec.With(prometheus.Labels{"Resolver": resolver.String(), "Operation": op}).
		Observe(startTimeToMS(start))
}

// SetGraphQLQueryDurationTime times how long a particular graphql API took on a particular resource.
func SetGraphQLQueryDurationTime(start time.Time, query string) {
	GraphQLQueryHistogramVec.With(prometheus.Labels{"Query": query}).Observe(startTimeToMS(start))
}

// SetSensorEventRunDuration times how long a particular sensor event operation took on a particular resource.
func SetSensorEventRunDuration(start time.Time, t, action string) {
	SensorEventDurationHistogramVec.With(prometheus.Labels{"Type": t, "Action": action}).Observe(startTimeToMS(start))
}

// SetIndexOperationDurationTime times how long a particular index operation took on a particular resource.
func SetIndexOperationDurationTime(start time.Time, op metrics.Op, t string) {
	IndexOperationHistogramVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).
		Observe(startTimeToMS(start))
}

// IncrementPipelinePanics increments the counter tracking the panics in pipeline processing.
func IncrementPipelinePanics(msg *central.MsgFromSensor) {
	resource := reflectutils.Type(msg.GetMsg())
	if event := msg.GetEvent(); event != nil {
		resource = reflectutils.Type(event.GetResource())
	}
	resource = stringutils.GetAfterLast(resource, "_")
	PipelinePanicCounter.With(prometheus.Labels{"resource": resource}).Inc()
}

// IncrementSensorConnect increments the counter for times that a new Sensor connection was observed.
func IncrementSensorConnect(clusterID, state string) {
	SensorConnectedCounter.With(prometheus.Labels{
		"ClusterID":        clusterID,
		"connection_state": state,
	}).Inc()
}

// IncrementSensorEventQueueCounter increments the counter for the passed operation.
func IncrementSensorEventQueueCounter(op metrics.Op, t string) {
	SensorEventQueueCounterVec.With(prometheus.Labels{"Operation": op.String(), "Type": t}).Inc()
}

// IncrementResourceProcessedCounter is a counter for how many times a resource has been processed in Central.
func IncrementResourceProcessedCounter(op metrics.Op, resource metrics.Resource) {
	ResourceProcessedCounterVec.With(prometheus.Labels{"Operation": op.String(), "Resource": resource.String()}).Inc()
}

// IncrementTotalNetworkFlowsReceivedCounter registers the total number of flows received.
func IncrementTotalNetworkFlowsReceivedCounter(clusterID string, numberOfFlows int) {
	TotalNetworkFlowsReceivedCounter.With(prometheus.Labels{"ClusterID": clusterID}).Add(float64(numberOfFlows))
}

// IncrementTotalNetworkEndpointsReceivedCounter registers the total number of endpoints received.
func IncrementTotalNetworkEndpointsReceivedCounter(clusterID string, numberOfEndpoints int) {
	TotalNetworkEndpointsReceivedCounter.With(prometheus.Labels{"ClusterID": clusterID}).Add(float64(numberOfEndpoints))
}

func IncrementTotalExternalPoliciesGauge() {
	TotalExternalPoliciesGauge.Inc()
	CurrentExternalPolicies.Inc()
}

func DecrementTotalExternalPoliciesGauge() {
	TotalExternalPoliciesGauge.Dec()
	CurrentExternalPolicies.Dec()
}

// ObserveRiskProcessingDuration adds an observation for risk processing duration.
func ObserveRiskProcessingDuration(startTime time.Time, riskObjectType string) {
	RiskProcessingHistogramVec.With(prometheus.Labels{"Risk_Reprocessor": riskObjectType}).
		Observe(startTimeToMS(startTime))
}

// SetDatastoreFunctionDuration is a histogram for datastore function timing.
func SetDatastoreFunctionDuration(start time.Time, resourceType, function string) {
	DatastoreFunctionDurationHistogramVec.With(prometheus.Labels{"Type": resourceType, "Function": function}).
		Observe(startTimeToMS(start))
}

// SetFunctionSegmentDuration times a specific segment within a function.
func SetFunctionSegmentDuration(start time.Time, segment string) {
	FunctionSegmentDurationHistogramVec.With(prometheus.Labels{"Segment": segment}).Observe(startTimeToMS(start))
}

// SetResourceProcessingDuration is the duration from sensor ingestion to Central processing.
func SetResourceProcessingDuration(event *central.SensorEvent) {
	metrics.SetResourceProcessingDurationForEvent(K8sObjectProcessingDuration, event, "")
}

// SetClusterMetrics sets cluster metrics to the values that have been collected by Sensor.
func SetClusterMetrics(clusterID string, clusterMetrics *central.ClusterMetrics) {
	ClusterMetricsNodeCountGaugeVec.With(prometheus.Labels{"ClusterID": clusterID}).
		Set(float64(clusterMetrics.GetNodeCount()))
	ClusterMetricsCPUCapacityGaugeVec.With(prometheus.Labels{"ClusterID": clusterID}).
		Set(float64(clusterMetrics.GetCpuCapacity()))
}

// IncrementOrphanedPLOPCounter increments the counter for orphaned PLOP
// objects. An orphaned PLOP objects indicates that something is not quite
// right, e.g. process information is received after the endpoint, or not
// received at all. This type of situations require investigation.
func IncrementOrphanedPLOPCounter(clusterID string) {
	TotalOrphanedPLOPCounter.With(prometheus.Labels{"ClusterID": clusterID}).Inc()
}

// ModifyProcessQueueLength modifies the metric for the number of processes that have not been flushed.
func ModifyProcessQueueLength(delta int) {
	ProcessQueueLengthGauge.Add(float64(delta))
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
	SensorEventsDeduperCounter.With(prometheus.Labels{"status": label, "type": typ}).Inc()
}

// SetReprocessorDuration registers how long a reprocessing step took.
func SetReprocessorDuration(start time.Time) {
	ReprocessorDurationGauge.Set(time.Since(start).Seconds())
}

// IncrementMsgToSensorNotSentCounter increments the count of messages not sent to Sensor due to
// errors or other reasons.
func IncrementMsgToSensorNotSentCounter(clusterID string, msg *central.MsgToSensor, reason string) {
	if msg.GetMsg() == nil {
		return
	}
	typ := event.GetEventTypeWithoutPrefix(msg.GetMsg())
	MsgToSensorNotSentCounter.With(prometheus.Labels{"ClusterID": clusterID, "type": typ, "reason": reason}).Inc()
}

// SetSignatureVerificationReprocessorDuration registers how long a signature verification reprocessing step took.
func SetSignatureVerificationReprocessorDuration(start time.Time) {
	SignatureVerificationReprocessorDurationGauge.Set(time.Since(start).Seconds())
}
