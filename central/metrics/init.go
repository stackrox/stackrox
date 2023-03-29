package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

func init() {
	// general

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		prometheus.MustRegister(
			boltOperationHistogramVec,
			rocksDBOperationHistogramVec,
			dackboxOperationHistogramVec,
		)
	}

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
