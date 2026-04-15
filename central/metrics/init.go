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

	// Scattered central metrics previously using init() are now called from their respective singleton/initialization functions:
	// - detection/lifecycle/metrics.Init() - called from detection/lifecycle/singleton.go
	// - detection/alertmanager.Init() - called from detection/alertmanager/singleton.go
	// - processindicator/datastore.Init() - called from processindicator/service/singleton.go
	// - complianceoperator/v2/report/manager.Init() - called from complianceoperator/v2/report/manager/singleton.go
	// - complianceoperator/v2/report/manager/watcher.Init() - called from complianceoperator/v2/report/manager/singleton.go
	// - image/datastore/store/common/v2.Init() - called from image/datastore/singleton.go
	// - imagev2/datastore/store/common.Init() - called from imagev2/datastore/singleton.go
	// - globaldb/metrics.Init() - called from globaldb/postgres.go InitializePostgres()
	// - hash/manager.Init() - called from hash/manager/singleton.go
	// - baseimage/watcher.Init() - called from baseimage/watcher/singleton.go
	// - sensor/service/connection/upgradecontroller.Init() - called from upgradecontroller/upgrade_controller.go New()
	// - sensor/service/pipeline/reprocessing.InitMetrics() - called from reprocessing/pipeline.go GetPipeline()
}
