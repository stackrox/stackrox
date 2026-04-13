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

const (
	ComponentName = "ComponentName"
	Operation     = "Operation"
)

var (
	// PanicCounter counts panics encountered
	PanicCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "panic_counter",
		Help:      "Number of panic calls within Sensor.",
	}, []string{"FunctionName"})

	DetectorDedupeCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_dedupe_cache_hits",
		Help:      "A counter of the total number of times we've deduped deployments in the detector",
	})

	DetectorDeploymentProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_deployment_processed",
		Help:      "A counter of the total number of times we've processed deployments in the detector",
	})

	ProcessDedupeCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "dedupe_cache_hits",
		Help:      "A counter of the total number of times we've deduped the process passed in",
	})

	ProcessDedupeCacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "dedupe_cache_misses",
		Help:      "A counter of the total number of times we've passed through the dedupe cache",
	})

	ProcessEnrichmentDrops = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_enrichment_drops",
		Help:      "Count of process indicators dropped because container metadata was not available before LRU eviction",
	})

	ProcessEnrichmentHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_enrichment_hits",
		Help:      "Count of process indicators successfully enriched with container metadata",
	})

	ProcessEnrichmentLRUCacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_enrichment_cache_size",
		Help:      "Current number of container entries waiting in the process-enrichment LRU cache",
	})

	SensorIndicatorChannelFullCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "indicators_channel_indicator_dropped_counter",
		Help:      "Total process indicator events dropped because the outgoing buffer to Central was full",
	})

	NetworkFlowBufferGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_buffer_size",
		Help:      "A gauge of the current size of the Network Flow buffer in Sensor (updated every 30s)",
	})

	EntitiesNotFound = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_network_flow_entity_not_found",
		Help:      "Total number of entities not found when processing Network Flows",
	}, []string{"kind", "orientation"})

	TotalNetworkFlowsReceivedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "total_network_flows_sensor_received_counter",
		Help:      "A counter of the total number of network flows received by Sensor from Collector",
	})

	ProcessSignalBufferGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_signal_buffer_size",
		Help:      "A gauge of the current size of the Process Indicator buffer in Sensor",
	})

	ProcessSignalDroppedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_signal_dropper_counter",
		Help:      "Count of process signals dropped due to shutdown or a full output buffer",
	})

	ProcessPipelineModeGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_pipeline_mode",
		Help:      "Indicates the active process pipeline mode (1 for the active mode, 0 for inactive)",
	}, []string{"mode"})

	SensorEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "sensor_events",
		Help:      "Total number of events sent from Sensor to Central",
	}, []string{"Action", "ResourceType", "Type"})

	SensorLastMessageSizeSent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "grpc_last_message_size_sent_bytes",
		Help:      "A gauge for last message size sent per message type",
	}, []string{"Type"})

	SensorMaxMessageSizeSent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "grpc_max_message_size_sent_bytes",
		Help:      "A gauge for maximum message size sent in the lifetime of this sensor",
	}, []string{"Type"})

	SensorMessageSizeSent = prometheus.NewHistogramVec(prometheus.HistogramOpts{
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
		},
	}, []string{"Type"})

	K8sObjectCounts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "k8s_events",
		Help:      "Total number of Kubernetes resource events processed by the Sensor listener",
	}, []string{"Action", "Resource"})

	ResourcesSyncedUnchaged = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "resources_synced_unchanged",
		Help:      "A counter to track how many resources were sent in ResourcesSynced message as stub ids",
	})

	ResourcesSyncedMessageSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "resources_synced_size",
		Help:      "Size in bytes of the most recent ResourcesSynced message sent to Central",
	})

	DeploymentEnhancementQueueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "deployment_enhancement_queue_size",
		Help:      "Current number of deployment enhancement requests from Central waiting to be processed",
	})

	K8sObjectIngestionToSendDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "k8s_event_ingestion_to_send_duration",
		Help:      "Sensor-side time from ingesting a Kubernetes event to sending the resulting update to Central in milliseconds",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 10),
	}, []string{"Action", "Resource", "Dispatcher", "Type"})

	K8sObjectProcessingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "k8s_event_processing_duration",
		Help:      "Time spent fully processing an event from Kubernetes in milliseconds",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 10),
	}, []string{"Action", "Resource", "Dispatcher"})

	ResolverChannelSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "resolver_channel_size",
		Help:      "Current number of resource events waiting in the resolver input queue",
	})

	// ResolverDedupingQueueSize a gauge to track the resolver's deduping queue size.
	ResolverDedupingQueueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "resolver_deduping_queue_size",
		Help:      "Current number of pending deployment references in the resolver deduping queue",
	})

	OutputChannelSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "output_channel_size",
		Help:      "Current number of resolved events waiting in the output queue before detector/forwarding",
	})

	telemetryLabels = prometheus.Labels{
		"branding":       branding.GetProductNameShort(),
		"build":          metrics.GetBuildType(),
		"sensor_version": version.GetMainVersion(),
	}

	TelemetryInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   metrics.PrometheusNamespace,
			Subsystem:   metrics.SensorSubsystem.String(),
			Name:        "info",
			Help:        "Telemetry information about Sensor",
			ConstLabels: telemetryLabels,
		},
		[]string{"central_id", "hosting", "install_method", "sensor_id"},
	)

	TelemetrySecuredNodes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   metrics.PrometheusNamespace,
			Subsystem:   metrics.SensorSubsystem.String(),
			Name:        "secured_nodes",
			Help:        "The number of nodes secured by Sensor",
			ConstLabels: telemetryLabels,
		},
		[]string{"central_id", "hosting", "install_method", "sensor_id"},
	)

	TelemetrySecuredVCPU = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   metrics.PrometheusNamespace,
			Subsystem:   metrics.SensorSubsystem.String(),
			Name:        "secured_vcpus",
			Help:        "The number of vCPUs secured by Sensor",
			ConstLabels: telemetryLabels,
		},
		[]string{"central_id", "hosting", "install_method", "sensor_id"},
	)

	TelemetryComplianceOperatorVersion = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   metrics.PrometheusNamespace,
			Subsystem:   metrics.SensorSubsystem.String(),
			Name:        "compliance_operator_version_info",
			Help:        "Version of compliance operator reported in label with constant value of 1",
			ConstLabels: telemetryLabels,
		},
		[]string{"central_id", "hosting", "install_method", "sensor_id", "compliance_operator_version"},
	)

	// ResponsesChannelOperationCount tracks operations in the responses channel
	ResponsesChannelOperationCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "num_messages_waiting_for_transmission_to_central",
		Help:      "Counts enqueue, dequeue, and drop operations on the Sensor-to-Central buffered stream",
	}, []string{Operation, "MessageType"})

	// ComponentProcessMessageDurationSeconds tracks the duration of ProcessMessage calls for each component
	ComponentProcessMessageDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "component_process_message_duration_seconds",
		Help:      "Time spent handling a message from Central inside a Sensor component in seconds",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 16),
	}, []string{ComponentName})

	// ComponentQueueOperations keeps track of the operations of the component queue buffer.
	ComponentQueueOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "component_queue_operations_total",
		Help:      "A counter that tracks the number of ADD and REMOVE operations on the component buffer queue. Current size of the queue can be calculated by subtracting the number of remove operations from the add operations",
	}, []string{ComponentName, Operation})

	// ComponentProcessMessageErrorsCount tracks the number of errors during ProcessMessage calls for each component
	ComponentProcessMessageErrorsCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "component_process_message_errors_total",
		Help:      "Number of errors encountered while processing messages from Central in each sensor component",
	}, []string{ComponentName})

	// InformersRegisteredCurrent is the total number of Kubernetes informers registered by Sensor
	// during startup. Each informer watches a specific resource type (e.g., Deployments, Pods,
	// NetworkPolicies). This number is set during initialization and remains constant for the
	// lifetime of the listener.
	InformersRegisteredCurrent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "informers_registered_current",
		Help: "Total number of Kubernetes informers registered by Sensor during startup. " +
			"Each informer watches a specific resource type (e.g., Deployments, Pods, NetworkPolicies). " +
			"Their number may vary depending on the features enabled and the type of the cluster. " +
			"This number is set during initialization and remains constant for the lifetime of the listener.",
	})

	// InformersPendingCurrent is the number of Kubernetes informers that have not yet completed
	// their initial sync. During normal startup this drops from the total to zero as each informer
	// finishes loading existing resources from the API server. A value that stays non-zero for an
	// extended period indicates a stuck informer.
	InformersPendingCurrent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "informers_pending_current",
		Help: "Number of Kubernetes informers that have not yet completed their initial sync. " +
			"During normal startup this drops from the total number of informers registered to zero as " +
			"each informer finishes loading existing resources from the API server. " +
			"A value that stays non-zero for an extended period indicates a stuck informer.",
	})

	// InformerSyncDurationMs tracks the time each informer has spent syncing.
	InformerSyncDurationMs = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "informer_sync_duration_ms",
		Help:      "Time in milliseconds each informer has spent syncing. While the informer is still pending, this value is updated periodically and keeps increasing. Once the informer completes its initial sync, the value is set to the final sync duration and stops changing. Labeled by informer name (e.g., Deployments, Pods). A value that keeps growing indicates a stuck informer.",
	}, []string{"informer"})

	// InformerInitialObjectPopulationDurationSeconds tracks post-sync startup work per informer.
	InformerInitialObjectPopulationDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "informer_initial_object_population_duration_seconds",
		Help: "Time in seconds spent processing and dispatching the initial object snapshot after informer cache sync. " +
			"Measured per informer from PopulateInitialObjects start until completion. " +
			"High values indicate slow initial object processing, which can delay full listener readiness.",
		Buckets: []float64{
			0.1, 0.25, 0.5,
			1, 2.5, 5, 10,
			30, 60, 120, 300,
		},
	}, []string{"informer"})
)

// IncrementEntityNotFound increments an instance of entity not found
func IncrementEntityNotFound(kind, orientation string) {
	EntitiesNotFound.With(prometheus.Labels{
		"kind":        kind,
		"orientation": orientation,
	}).Inc()
}

// IncrementDetectorCacheHit increments the number of deployments deduped by the detector
func IncrementDetectorCacheHit() {
	DetectorDedupeCacheHits.Inc()
}

// IncrementDetectorDeploymentProcessed increments the number of deployments processed by the detector
func IncrementDetectorDeploymentProcessed() {
	DetectorDeploymentProcessed.Inc()
}

// IncrementPanicCounter increments the number of panic calls seen in a function
func IncrementPanicCounter(functionName string) {
	PanicCounter.With(prometheus.Labels{"FunctionName": functionName}).Inc()
}

// IncrementProcessDedupeCacheHits increments the number of times we deduped a process
func IncrementProcessDedupeCacheHits() {
	ProcessDedupeCacheHits.Inc()
}

// IncrementProcessDedupeCacheMisses increments the number of times we failed to dedupe a process
func IncrementProcessDedupeCacheMisses() {
	ProcessDedupeCacheMisses.Inc()
}

// RegisterSensorIndicatorChannelFullCounter registers the number of indicators dropped
func RegisterSensorIndicatorChannelFullCounter() {
	SensorIndicatorChannelFullCounter.Inc()
}

// IncrementDeploymentEnhancerQueueSize increments the deployment enhancer queue size by one.
func IncrementDeploymentEnhancerQueueSize() {
	DeploymentEnhancementQueueSize.Inc()
}

// DecrementDeploymentEnhancerQueueSize decrements the deployment enhancer queue size by one.
func DecrementDeploymentEnhancerQueueSize() {
	DeploymentEnhancementQueueSize.Dec()
}

// IncrementTotalResourcesSyncSent sets the number of resources synced transmitted in the last sync event
func IncrementTotalResourcesSyncSent(value int) {
	ResourcesSyncedUnchaged.Add(float64(value))
}

// SetResourcesSyncedSize sets the latest resources synced message size transmitted to central.
func SetResourcesSyncedSize(size int) {
	ResourcesSyncedMessageSize.Set(float64(size))
}

// SetNetworkFlowBufferSizeGauge set network flow buffer size gauge.
func SetNetworkFlowBufferSizeGauge(v int) {
	NetworkFlowBufferGauge.Set(float64(v))
}

// IncrementTotalNetworkFlowsReceivedCounter registers the total number of flows received
func IncrementTotalNetworkFlowsReceivedCounter(numberOfFlows int) {
	TotalNetworkFlowsReceivedCounter.Add(float64(numberOfFlows))
}

// SetProcessSignalBufferSizeGauge set process signal buffer size gauge.
func SetProcessSignalBufferSizeGauge(number int) {
	ProcessSignalBufferGauge.Set(float64(number))
}

// IncrementProcessSignalDroppedCount increments the number of times the process signal was dropped.
func IncrementProcessSignalDroppedCount() {
	ProcessSignalDroppedCount.Inc()
}

// IncrementProcessEnrichmentDrops increments the number of times we could not enrich.
func IncrementProcessEnrichmentDrops() {
	ProcessEnrichmentDrops.Inc()
}

// IncrementProcessEnrichmentHits increments the number of times we could enrich.
func IncrementProcessEnrichmentHits() {
	ProcessEnrichmentHits.Inc()
}

// SetProcessEnrichmentCacheSize sets the enrichment cache size.
func SetProcessEnrichmentCacheSize(size float64) {
	ProcessEnrichmentLRUCacheSize.Set(size)
}

const (
	// ProcessPipelineModePubSub indicates the pub/sub pipeline mode is active.
	ProcessPipelineModePubSub = "pubsub"
	// ProcessPipelineModeLegacy indicates the legacy channel pipeline mode is active.
	ProcessPipelineModeLegacy = "legacy"
)

// SetProcessPipelineMode sets which process pipeline mode is active.
func SetProcessPipelineMode(mode string) {
	ProcessPipelineModeGauge.Reset()
	ProcessPipelineModeGauge.WithLabelValues(mode).Set(1)
}

// IncK8sEventCount increments the number of objects we're receiving from k8s
func IncK8sEventCount(action string, resource string) {
	K8sObjectCounts.With(prometheus.Labels{
		"Action":   action,
		"Resource": resource,
	}).Inc()
}

// SetResourceProcessingDurationForResource sets the duration for how long it takes to process the resource
func SetResourceProcessingDurationForResource(event *central.SensorEvent) {
	metrics.SetResourceProcessingDurationForEvent(K8sObjectProcessingDuration, event, "")
}

// IncResolverChannelSize increases the resolverChannel by 1
func IncResolverChannelSize() {
	ResolverChannelSize.Inc()
}

// DecResolverChannelSize decreases the resolverChannel by 1
func DecResolverChannelSize() {
	ResolverChannelSize.Dec()
}

// IncOutputChannelSize increases the outputChannel by 1
func IncOutputChannelSize() {
	OutputChannelSize.Inc()
}

// DecOutputChannelSize decreases the outputChannel by 1
func DecOutputChannelSize() {
	OutputChannelSize.Dec()
}

func getResponsesChannelLabel(op string, msg *central.MsgFromSensor) prometheus.Labels {
	msgType := "nil"
	if msg.GetMsg() != nil {
		msgType = strings.TrimPrefix(reflect.TypeOf(msg.GetMsg()).String(), "*central.MsgFromSensor_")
	}
	return prometheus.Labels{
		"MessageType": msgType,
		Operation:     op,
	}
}

// ResponsesChannelAdd increases the ResponsesChannelOperationCount's Add operation by 1
func ResponsesChannelAdd(msg *central.MsgFromSensor) {
	ResponsesChannelOperationCount.With(getResponsesChannelLabel(metrics.Add.String(), msg)).Inc()
}

// ResponsesChannelRemove increases the ResponsesChannelOperationCount's Remove operation by 1
func ResponsesChannelRemove(msg *central.MsgFromSensor) {
	ResponsesChannelOperationCount.With(getResponsesChannelLabel(metrics.Remove.String(), msg)).Inc()
}

// ResponsesChannelDrop increases the responsesChannelDroppedCount by 1
func ResponsesChannelDrop(msg *central.MsgFromSensor) {
	ResponsesChannelOperationCount.With(getResponsesChannelLabel(metrics.Dropped.String(), msg)).Inc()
}

// SetTelemetryMetrics sets the cluster metrics for the telemetry metrics.
func SetTelemetryMetrics(clusterIDPeeker func() string, cm *central.ClusterMetrics) {
	labels := []string{
		centralid.Get(),
		getHosting(),
		installmethod.Get(),
		clusterIDPeeker(),
	}

	TelemetryInfo.Reset()
	TelemetryInfo.WithLabelValues(labels...).Set(1)

	TelemetrySecuredNodes.Reset()
	TelemetrySecuredNodes.WithLabelValues(labels...).Set(float64(cm.GetNodeCount()))

	TelemetrySecuredVCPU.Reset()
	TelemetrySecuredVCPU.WithLabelValues(labels...).Set(float64(cm.GetCpuCapacity()))

	TelemetryComplianceOperatorVersion.Reset()
	TelemetryComplianceOperatorVersion.WithLabelValues(append(labels, cm.GetComplianceOperatorVersion())...).Set(1)
}

// ObserveCentralReceiverProcessMessageDuration records the duration of a ProcessMessage call
func ObserveCentralReceiverProcessMessageDuration(componentName string, duration time.Duration) {
	ComponentProcessMessageDurationSeconds.With(prometheus.Labels{
		ComponentName: componentName,
	}).Observe(duration.Seconds())
}

// IncrementCentralReceiverProcessMessageErrors increments the error count for a component's ProcessMessage call
func IncrementCentralReceiverProcessMessageErrors(componentName string) {
	ComponentProcessMessageErrorsCount.With(prometheus.Labels{
		ComponentName: componentName,
	}).Inc()
}

// ObserveInformerSyncDuration sets the sync duration metric for an informer.
// Called periodically for pending informers (with the elapsed time so far)
// and once for completed informers (with the final sync duration).
func ObserveInformerSyncDuration(informerName string, duration time.Duration) {
	InformerSyncDurationMs.WithLabelValues(informerName).Set(float64(duration.Milliseconds()))
}

// ResetInformerSyncDuration removes all label values from the sync duration gauge,
// clearing stale per-informer entries from a previous tracker lifecycle.
func ResetInformerSyncDuration() {
	InformerSyncDurationMs.Reset()
}

// ObserveInformerInitialObjectPopulationDuration records how long initial object population took for an informer.
func ObserveInformerInitialObjectPopulationDuration(informerName string, duration time.Duration) {
	InformerInitialObjectPopulationDurationSeconds.WithLabelValues(informerName).Observe(duration.Seconds())
}
