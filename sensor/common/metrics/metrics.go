package metrics

import (
	"reflect"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/common/centralid"
	"github.com/stackrox/rox/sensor/common/installmethod"
)

var (
	// Panics encountered
	panicCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "panic_counter",
		Help:      "Number of panic calls within Sensor.",
	}, []string{"FunctionName"})

	detectorDedupeCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_dedupe_cache_hits",
		Help:      "A counter of the total number of times we've deduped deployments in the detector",
	})

	detectorDeploymentProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_deployment_processed",
		Help:      "A counter of the total number of times we've processed deployments in the detector",
	})

	processDedupeCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "dedupe_cache_hits",
		Help:      "A counter of the total number of times we've deduped the process passed in",
	})

	processDedupeCacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "dedupe_cache_misses",
		Help:      "A counter of the total number of times we've passed through the dedupe cache",
	})

	processEnrichmentDrops = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_enrichment_drops",
		Help:      "A counter of the total number of times we've dropped enriching process indicators",
	})

	processEnrichmentHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_enrichment_hits",
		Help:      "A counter of the total number of times we've successfully enriched process indicators",
	})

	processEnrichmentLRUCacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_enrichment_cache_size",
		Help:      "A gauge to track the enrichment lru cache size",
	})

	sensorIndicatorChannelFullCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "indicators_channel_indicator_dropped_counter",
		Help:      "A counter of the total number of times we've dropped indicators from the indicators channel because it was full",
	})

	networkFlowBufferGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_buffer_size",
		Help:      "A gauge of the current size of the Network Flow buffer in Sensor (updated every 30s)",
	})

	entitiesNotFound = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_network_flow_entity_not_found",
		Help:      "Total number of entities not found when processing Network Flows",
	}, []string{"kind", "orientation"})

	totalNetworkFlowsReceivedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "total_network_flows_sensor_received_counter",
		Help:      "A counter of the total number of network flows received by Sensor from Collector",
	})

	processSignalBufferGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_signal_buffer_size",
		Help:      "A gauge of the current size of the Process Indicator buffer in Sensor",
	})

	processSignalDroppedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_signal_dropper_counter",
		Help:      "A counter of the total number of process indicators that were dropped if the buffer was full",
	})

	sensorEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "sensor_events",
		Help:      "A counter for the total number of events sent from Sensor to Central",
	}, []string{"Action", "ResourceType", "Type"})

	sensorLastMessageSizeSent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "grpc_last_message_size_sent_bytes",
		Help:      "A gauge for last message size sent per message type",
	}, []string{"Type"})

	sensorMaxMessageSizeSent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "grpc_max_message_size_sent_bytes",
		Help:      "A gauge for maximum message size sent in the lifetime of this sensor",
	}, []string{"Type"})

	sensorMessageSizeSent = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "grpc_message_size_sent_bytes",
		Help:      "A histogram for message sizes sent by sensor",
		Buckets: []float64{
			4_000_000,
			12_000_000,
			24_000_000,
			48_000_000,
			256_000_000,
		}, // Bucket sizes selected arbitrary based on current default limits for grpc message size
	}, []string{"Type"})

	k8sObjectCounts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "k8s_events",
		Help:      "A counter for the total number of typed k8s events processed by Sensor",
	}, []string{"Action", "Resource"})

	resourcesSyncedUnchaged = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "resources_synced_unchanged",
		Help:      "A counter to track how many resources were sent in ResourcesSynced message as stub ids",
	})

	resourcesSyncedMessageSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "resources_synced_size",
		Help:      "A gauge to track how large ResourcesSynced message is",
	})

	deploymentEnhancementQueueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "deployment_enhancement_queue_size",
		Help:      "A counter to track deployments queued up in Sensor to be enhanced",
	})

	k8sObjectIngestionToSendDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "k8s_event_ingestion_to_send_duration",
		Help:      "Time taken to fully process an event from Kubernetes",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Action", "Resource", "Dispatcher", "Type"})

	k8sObjectProcessingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "k8s_event_processing_duration",
		Help:      "Time taken to fully process an event from Kubernetes",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Action", "Resource", "Dispatcher"})

	resolverChannelSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "resolver_channel_size",
		Help:      "A gauge to track the resolver channel size",
	})

	// ResolverDedupingQueueSize a gauge to track the resolver's deduping queue size.
	ResolverDedupingQueueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "resolver_deduping_queue_size",
		Help:      "A gauge to track the resolver deduping queue size",
	})

	outputChannelSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "output_channel_size",
		Help:      "A gauge to track the output channel size",
	})

	telemetryLabels = prometheus.Labels{
		"branding":       branding.GetProductNameShort(),
		"build":          metrics.GetBuildType(),
		"sensor_version": version.GetMainVersion(),
	}

	telemetryInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   metrics.PrometheusNamespace,
			Subsystem:   metrics.SensorSubsystem.String(),
			Name:        "info",
			Help:        "Telemetry information about Sensor",
			ConstLabels: telemetryLabels,
		},
		[]string{"central_id", "hosting", "install_method", "sensor_id"},
	)

	telemetrySecuredNodes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   metrics.PrometheusNamespace,
			Subsystem:   metrics.SensorSubsystem.String(),
			Name:        "secured_nodes",
			Help:        "The number of nodes secured by Sensor",
			ConstLabels: telemetryLabels,
		},
		[]string{"central_id", "hosting", "install_method", "sensor_id"},
	)

	telemetrySecuredVCPU = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   metrics.PrometheusNamespace,
			Subsystem:   metrics.SensorSubsystem.String(),
			Name:        "secured_vcpus",
			Help:        "The number of vCPUs secured by Sensor",
			ConstLabels: telemetryLabels,
		},
		[]string{"central_id", "hosting", "install_method", "sensor_id"},
	)

	// responsesChannelOperationCount a counter to track the operations in the responses channel
	responsesChannelOperationCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "num_messages_waiting_for_transmission_to_central",
		Help:      "A counter that tracks the operations in the responses channel",
	}, []string{"Operation", "MessageType"})

	// componentProcessMessageDurationSeconds tracks the duration of ProcessMessage calls for each component
	componentProcessMessageDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "component_process_message_duration_seconds",
		Help:      "Time taken to process messages from Central in each sensor component",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to ~4s
	}, []string{"ComponentName"})

	// ComponentQueueMessagesDroppedCount tracks the number of messages dropped due to context timeout
	ComponentQueueMessagesDroppedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "component_queue_messages_dropped_count",
		Help:      "Number of messages dropped due to full queue",
	})

	// ComponentQueueOperations keeps track of the operations of the component queue buffer.
	ComponentQueueOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "component_queue_operations_total",
		Help:      "A counter that tracks the number of ADD and REMOVE operations on the component buffer queue. Current size of the queue can be calculated by subtracting the number of remove operations from the add operations",
	}, []string{"Operation"})
)

// IncrementEntityNotFound increments an instance of entity not found
func IncrementEntityNotFound(kind, orientation string) {
	entitiesNotFound.With(prometheus.Labels{
		"kind":        kind,
		"orientation": orientation,
	}).Inc()
}

// IncrementDetectorCacheHit increments the number of deployments deduped by the detector
func IncrementDetectorCacheHit() {
	detectorDedupeCacheHits.Inc()
}

// IncrementDetectorDeploymentProcessed increments the number of deployments processed by the detector
func IncrementDetectorDeploymentProcessed() {
	detectorDeploymentProcessed.Inc()
}

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

// RegisterSensorIndicatorChannelFullCounter registers the number of indicators dropped
func RegisterSensorIndicatorChannelFullCounter() {
	sensorIndicatorChannelFullCounter.Inc()
}

// IncrementDeploymentEnhancerQueueSize increments the deployment enhancer queue size by one.
func IncrementDeploymentEnhancerQueueSize() {
	deploymentEnhancementQueueSize.Inc()
}

// DecrementDeploymentEnhancerQueueSize decrements the deployment enhancer queue size by one.
func DecrementDeploymentEnhancerQueueSize() {
	deploymentEnhancementQueueSize.Dec()
}

// IncrementTotalResourcesSyncSent sets the number of resources synced transmitted in the last sync event
func IncrementTotalResourcesSyncSent(value int) {
	resourcesSyncedUnchaged.Add(float64(value))
}

// SetResourcesSyncedSize sets the latest resources synced message size transmitted to central.
func SetResourcesSyncedSize(size int) {
	resourcesSyncedMessageSize.Set(float64(size))
}

// SetNetworkFlowBufferSizeGauge set network flow buffer size gauge.
func SetNetworkFlowBufferSizeGauge(v int) {
	networkFlowBufferGauge.Set(float64(v))
}

// IncrementTotalNetworkFlowsReceivedCounter registers the total number of flows received
func IncrementTotalNetworkFlowsReceivedCounter(numberOfFlows int) {
	totalNetworkFlowsReceivedCounter.Add(float64(numberOfFlows))
}

// SetProcessSignalBufferSizeGauge set process signal buffer size gauge.
func SetProcessSignalBufferSizeGauge(number int) {
	processSignalBufferGauge.Set(float64(number))
}

// IncrementProcessSignalDroppedCount increments the number of times the process signal was dropped.
func IncrementProcessSignalDroppedCount() {
	processSignalDroppedCount.Inc()
}

// IncrementProcessEnrichmentDrops increments the number of times we could not enrich.
func IncrementProcessEnrichmentDrops() {
	processEnrichmentDrops.Inc()
}

// IncrementProcessEnrichmentHits increments the number of times we could enrich.
func IncrementProcessEnrichmentHits() {
	processEnrichmentHits.Inc()
}

// SetProcessEnrichmentCacheSize sets the enrichment cache size.
func SetProcessEnrichmentCacheSize(size float64) {
	processEnrichmentLRUCacheSize.Set(size)
}

// IncK8sEventCount increments the number of objects we're receiving from k8s
func IncK8sEventCount(action string, resource string) {
	k8sObjectCounts.With(prometheus.Labels{
		"Action":   action,
		"Resource": resource,
	}).Inc()
}

// SetResourceProcessingDurationForResource sets the duration for how long it takes to process the resource
func SetResourceProcessingDurationForResource(event *central.SensorEvent) {
	metrics.SetResourceProcessingDurationForEvent(k8sObjectProcessingDuration, event, "")
}

// IncResolverChannelSize increases the resolverChannel by 1
func IncResolverChannelSize() {
	resolverChannelSize.Inc()
}

// DecResolverChannelSize decreases the resolverChannel by 1
func DecResolverChannelSize() {
	resolverChannelSize.Dec()
}

// IncOutputChannelSize increases the outputChannel by 1
func IncOutputChannelSize() {
	outputChannelSize.Inc()
}

// DecOutputChannelSize decreases the outputChannel by 1
func DecOutputChannelSize() {
	outputChannelSize.Dec()
}

func getResponsesChannelLabel(op string, msg *central.MsgFromSensor) prometheus.Labels {
	msgType := "nil"
	if msg.GetMsg() != nil {
		msgType = strings.TrimPrefix(reflect.TypeOf(msg.GetMsg()).String(), "*central.MsgFromSensor_")
	}
	return prometheus.Labels{
		"MessageType": msgType,
		"Operation":   op,
	}
}

// ResponsesChannelAdd increases the responsesChannelOperationCount's Add operation by 1
func ResponsesChannelAdd(msg *central.MsgFromSensor) {
	responsesChannelOperationCount.With(getResponsesChannelLabel(metrics.Add.String(), msg)).Inc()
}

// ResponsesChannelRemove increases the responsesChannelOperationCount's Remove operation by 1
func ResponsesChannelRemove(msg *central.MsgFromSensor) {
	responsesChannelOperationCount.With(getResponsesChannelLabel(metrics.Remove.String(), msg)).Inc()
}

// ResponsesChannelDrop increases the responsesChannelDroppedCount by 1
func ResponsesChannelDrop(msg *central.MsgFromSensor) {
	responsesChannelOperationCount.With(getResponsesChannelLabel(metrics.Dropped.String(), msg)).Inc()
}

// SetTelemetryMetrics sets the cluster metrics for the telemetry metrics.
func SetTelemetryMetrics(clusterIDPeeker func() string, cm *central.ClusterMetrics) {
	labels := []string{
		centralid.Get(),
		getHosting(),
		installmethod.Get(),
		clusterIDPeeker(),
	}

	telemetryInfo.Reset()
	telemetryInfo.WithLabelValues(labels...).Set(1)

	telemetrySecuredNodes.Reset()
	telemetrySecuredNodes.WithLabelValues(labels...).Set(float64(cm.GetNodeCount()))

	telemetrySecuredVCPU.Reset()
	telemetrySecuredVCPU.WithLabelValues(labels...).Set(float64(cm.GetCpuCapacity()))
}

// ObserveCentralReceiverProcessMessageDuration records the duration of a ProcessMessage call
func ObserveCentralReceiverProcessMessageDuration(componentName string, duration time.Duration) {
	componentProcessMessageDurationSeconds.With(prometheus.Labels{
		"ComponentName": componentName,
	}).Observe(duration.Seconds())
}
