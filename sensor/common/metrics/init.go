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
		totalNetworkFlowsSentCounter,
		totalNetworkFlowsReceivedCounter,
		totalNetworkEndpointsSentCounter,
		totalNetworkEndpointsReceivedCounter,
		totalProcessesSentCounter,
		totalProcessesReceivedCounter,
		sensorEvents,
		k8sObjectCounts,
		k8sObjectProcessingDuration,
		k8sObjectIngestionToSendDuration,
		resolverChannelSize,
		resourcesSyncedUnchaged,
		outputChannelSize,
		telemetryInfo,
		telemetrySecuredNodes,
		telemetrySecuredVCPU,
	)
}
