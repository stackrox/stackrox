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
		policyEvaluationHistogram,
		resourceProcessedCounterVec,
		totalNetworkFlowsReceivedCounter,
		totalNetworkEndpointsReceivedCounter,
		sensorEventDurationHistogramVec,
		riskProcessingHistogramVec,
		totalCacheOperationsCounter,
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
	)
}
