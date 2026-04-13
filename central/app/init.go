package app

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/metrics"
)

func initMetrics() {
	prometheus.MustRegister(
		metrics.PipelinePanicCounter,
		metrics.GraphQLOperationHistogramVec,
		metrics.GraphQLQueryHistogramVec,
		metrics.IndexOperationHistogramVec,
		metrics.SensorEventQueueCounterVec,
		metrics.ResourceProcessedCounterVec,
		metrics.TotalNetworkFlowsReceivedCounter,
		metrics.TotalNetworkEndpointsReceivedCounter,
		metrics.TotalExternalPoliciesGauge,
		metrics.CurrentExternalPolicies,
		metrics.SensorEventDurationHistogramVec,
		metrics.RiskProcessingHistogramVec,
		metrics.DatastoreFunctionDurationHistogramVec,
		metrics.FunctionSegmentDurationHistogramVec,
		metrics.K8sObjectProcessingDuration,
		metrics.PostgresOperationHistogramVec,
		metrics.AcquireDBConnHistogramVec,
		metrics.ClusterMetricsNodeCountGaugeVec,
		metrics.ClusterMetricsCPUCapacityGaugeVec,
		metrics.TotalOrphanedPLOPCounter,
		metrics.ProcessQueueLengthGauge,
		metrics.SensorEventsDeduperCounter,
		metrics.SensorConnectedCounter,
		metrics.GrpcMaxMessageSize,
		metrics.GrpcSentSize,
		metrics.GrpcLastMessageSizeSent,
		metrics.GrpcLastMessageSizeReceived,
		metrics.GrpcError,
		metrics.DeploymentEnhancementRoundTripDuration,
		metrics.ReprocessorDurationGauge,
		metrics.SignatureVerificationReprocessorDurationGauge,
		metrics.PruningDurationHistogramVec,
		metrics.StoreCacheOperationHistogramVec,
		metrics.MsgToSensorNotSentCounter,
	)
}

// initCompliance registers all compliance checks.
func initCompliance() {
	// Import side-effect: pkg/compliance/checks registers all standard checks via init()
	// We consolidate that registration here by calling the registration function explicitly
	// This is handled by importing central/compliance/checks/remote which calls MustRegisterChecks

	// The actual registration is done via the package import
	// Future work: refactor compliance/checks to use explicit registration
}

// initGraphQL registers all GraphQL type loaders.
func initGraphQL() {
	// GraphQL loaders registration
	// Each loader registers itself via RegisterTypeFactory in their init() functions

	// Similar to compliance checks, this requires refactoring the loader registration
	// to be explicit rather than init()-based
	// Stub for now - full migration in separate PR
}
