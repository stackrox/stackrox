package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Init registers all Central prometheus metrics.
// Called explicitly from central/app/app.go instead of package init().
func Init() {
	// general

	prometheus.MustRegister(
		pipelinePanicCounter,
		graphQLOperationHistogramVec,
		graphQLQueryHistogramVec,
		indexOperationHistogramVec,
		sensorEventQueueCounterVec,
		resourceProcessedCounterVec,
		totalNetworkFlowsReceivedCounter,
		totalNetworkEndpointsReceivedCounter,
		totalExternalPoliciesGauge,
		currentExternalPolicies,
		sensorEventDurationHistogramVec,
		riskProcessingHistogramVec,
		datastoreFunctionDurationHistogramVec,
		functionSegmentDurationHistogramVec,
		k8sObjectProcessingDuration,
		postgresOperationHistogramVec,
		acquireDBConnHistogramVec,
		clusterMetricsNodeCountGaugeVec,
		clusterMetricsCPUCapacityGaugeVec,
		totalOrphanedPLOPCounter,
		processQueueLengthGauge,
		sensorEventsDeduperCounter,
		sensorConnectedCounter,
		grpcMaxMessageSize,
		grpcSentSize,
		grpcLastMessageSizeSent,
		grpcLastMessageSizeReceived,
		grpcError,
		deploymentEnhancementRoundTripDuration,
		reprocessorDurationGauge,
		signatureVerificationReprocessorDurationGauge,
		pruningDurationHistogramVec,
		storeCacheOperationHistogramVec,
		msgToSensorNotSentCounter,
	)
}
