package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Init registers all Sensor prometheus metrics.
// Called explicitly from sensor/kubernetes/app/app.go instead of package init().
func Init() {
	// general
	prometheus.MustRegister(
		panicCounter,
		detectorDedupeCacheHits,
		detectorDeploymentProcessed,
		processDedupeCacheHits,
		processDedupeCacheMisses,
		processEnrichmentHits,
		processEnrichmentDrops,
		processEnrichmentLRUCacheSize,
		sensorIndicatorChannelFullCounter,
		networkFlowBufferGauge,
		entitiesNotFound,
		totalNetworkFlowsReceivedCounter,
		processSignalBufferGauge,
		processSignalDroppedCount,
		processPipelineModeGauge,
		sensorEvents,
		sensorMaxMessageSizeSent,
		sensorMessageSizeSent,
		sensorLastMessageSizeSent,
		k8sObjectCounts,
		k8sObjectProcessingDuration,
		k8sObjectIngestionToSendDuration,
		resolverChannelSize,
		ResolverDedupingQueueSize,
		resourcesSyncedUnchaged,
		resourcesSyncedMessageSize,
		outputChannelSize,
		telemetryInfo,
		telemetrySecuredNodes,
		telemetrySecuredVCPU,
		telemetryComplianceOperatorVersion,
		deploymentEnhancementQueueSize,
		responsesChannelOperationCount,
		ComponentQueueOperations,
		componentProcessMessageDurationSeconds,
		componentProcessMessageErrorsCount,
		InformersRegisteredCurrent,
		InformersPendingCurrent,
		informerSyncDurationMs,
		informerInitialObjectPopulationDurationSeconds,
	)
}
