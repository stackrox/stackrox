package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

func init() {
	// general
	prometheus.MustRegister(
		boltOperationHistogramVec,
		rocksDBOperationHistogramVec,
		dackboxOperationHistogramVec,
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
	)
}
