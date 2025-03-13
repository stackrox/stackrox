package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
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
		totalPolicyAsCodeCRsReceivedGauge,
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
	)
}
