package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	clusterentitiesmetrics "github.com/stackrox/rox/sensor/common/clusterentities/metrics"
	detectormetrics "github.com/stackrox/rox/sensor/common/detector/metrics"
	networkflowmetrics "github.com/stackrox/rox/sensor/common/networkflow/metrics"
	updatecomputermetrics "github.com/stackrox/rox/sensor/common/networkflow/updatecomputer"
	pubsubmetrics "github.com/stackrox/rox/sensor/common/pubsub/metrics"
	registrymetrics "github.com/stackrox/rox/sensor/common/registry/metrics"
	virtualmachinemetrics "github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
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

	// component-specific metrics
	clusterentitiesmetrics.Init()
	detectormetrics.Init()
	networkflowmetrics.Init()
	updatecomputermetrics.Init()
	pubsubmetrics.Init()
	registrymetrics.Init()
	virtualmachinemetrics.Init()
}
