package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
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
		totalNetworkFlowsSentCounter,
		totalNetworkFlowsReceivedCounter,
		totalNetworkEndpointsSentCounter,
		totalNetworkEndpointsReceivedCounter,
		totalProcessesSentCounter,
		totalProcessesReceivedCounter,
		processSignalBufferGauge,
		processSignalDroppedCount,
		sensorEvents,
		sensorGRPCMaxMessageSize,
		k8sObjectCounts,
		k8sObjectProcessingDuration,
		k8sObjectIngestionToSendDuration,
		resolverChannelSize,
		resourcesSyncedUnchaged,
		resourcesSyncedMessageSize,
		outputChannelSize,
		telemetryInfo,
		telemetrySecuredNodes,
		telemetrySecuredVCPU,
	)
}
