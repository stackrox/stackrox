package app

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/sensor/common/metrics"
)

func initMetrics() {
	prometheus.MustRegister(
		metrics.PanicCounter,
		metrics.DetectorDedupeCacheHits,
		metrics.DetectorDeploymentProcessed,
		metrics.ProcessDedupeCacheHits,
		metrics.ProcessDedupeCacheMisses,
		metrics.ProcessEnrichmentHits,
		metrics.ProcessEnrichmentDrops,
		metrics.ProcessEnrichmentLRUCacheSize,
		metrics.SensorIndicatorChannelFullCounter,
		metrics.NetworkFlowBufferGauge,
		metrics.EntitiesNotFound,
		metrics.TotalNetworkFlowsReceivedCounter,
		metrics.ProcessSignalBufferGauge,
		metrics.ProcessSignalDroppedCount,
		metrics.ProcessPipelineModeGauge,
		metrics.SensorEvents,
		metrics.SensorMaxMessageSizeSent,
		metrics.SensorMessageSizeSent,
		metrics.SensorLastMessageSizeSent,
		metrics.K8sObjectCounts,
		metrics.K8sObjectProcessingDuration,
		metrics.K8sObjectIngestionToSendDuration,
		metrics.ResolverChannelSize,
		metrics.ResolverDedupingQueueSize,
		metrics.ResourcesSyncedUnchaged,
		metrics.ResourcesSyncedMessageSize,
		metrics.OutputChannelSize,
		metrics.TelemetryInfo,
		metrics.TelemetrySecuredNodes,
		metrics.TelemetrySecuredVCPU,
		metrics.TelemetryComplianceOperatorVersion,
		metrics.DeploymentEnhancementQueueSize,
		metrics.ResponsesChannelOperationCount,
		metrics.ComponentQueueOperations,
		metrics.ComponentProcessMessageDurationSeconds,
		metrics.ComponentProcessMessageErrorsCount,
		metrics.InformersRegisteredCurrent,
		metrics.InformersPendingCurrent,
		metrics.InformerSyncDurationMs,
		metrics.InformerInitialObjectPopulationDurationSeconds,
	)
}
