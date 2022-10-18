package centralsensor

const (
	// PullMetricsCap identifies the capability to pull metrics from sensor.
	PullMetricsCap SensorCapability = "PullMetrics"

	// PullTelemetryDataCap identifies the capability to pull telemetry data from sensor.
	PullTelemetryDataCap SensorCapability = "PullTelemetryData"

	// CancelTelemetryPullCap identifies the capability to cancel an ongoing telemetry data pull.
	CancelTelemetryPullCap SensorCapability = "CancelTelemetryPull"

	// SensorDetectionCap identifies the capability to run detection from sensor
	SensorDetectionCap SensorCapability = "SensorDetection"

	// ComplianceInNodesCap identifies the capability to run compliance in compliance pods
	ComplianceInNodesCap SensorCapability = "ComplianceInNodes"

	// HealthMonitoringCap identifies the capability to send health information
	HealthMonitoringCap SensorCapability = "HealthMonitoring"

	// NetworkGraphExternalSrcsCap identifies the capability to handle custom network graph external sources.
	NetworkGraphExternalSrcsCap SensorCapability = "NetworkGraphExternalSrcs"

	// AuditLogEventsCap identifies the capability to handle audit log event detection.
	AuditLogEventsCap SensorCapability = "AuditLogEvents"

	// LocalScannerCredentialsRefresh identifies the capability to maintain the Local scanner TLS credentials refreshed.
	LocalScannerCredentialsRefresh SensorCapability = "LocalScannerCredentialsRefresh"

	// ScopedImageIntegrations identifies the capability to have image integrations with sources from image pull secrets
	ScopedImageIntegrations SensorCapability = "ScopedImageIntegrations"
)
