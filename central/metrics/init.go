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
		badgerOperationHistogramVec,
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
		sensorEventDurationHistogramVec,
		riskProcessingHistogramVec,
		totalCacheOperationsCounter,
		datastoreFunctionDurationHistogramVec,
		functionSegmentDurationHistogramVec,
	)
}
