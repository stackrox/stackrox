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

	// SensorEnhancedDeploymentCheckCap identifies the capability for sensor to enhance roxctl deployment check
	SensorEnhancedDeploymentCheckCap SensorCapability = "SensorEnhancedDeploymentCheck"

	// ComplianceInNodesCap identifies the capability to run compliance in compliance pods
	ComplianceInNodesCap SensorCapability = "ComplianceInNodes"

	// HealthMonitoringCap identifies the capability to send health information
	HealthMonitoringCap SensorCapability = "HealthMonitoring"

	// NetworkGraphExternalSrcsCap identifies the capability to handle custom network graph external sources.
	NetworkGraphExternalSrcsCap SensorCapability = "NetworkGraphExternalSrcs" //#nosec G101

	// AuditLogEventsCap identifies the capability to handle audit log event detection.
	AuditLogEventsCap SensorCapability = "AuditLogEvents"

	// LocalScannerCredentialsRefresh identifies the capability to maintain the Local scanner TLS credentials refreshed.
	LocalScannerCredentialsRefresh SensorCapability = "LocalScannerCredentialsRefresh"

	// ScopedImageIntegrations identifies the capability to have image integrations with sources from image pull secrets
	ScopedImageIntegrations SensorCapability = "ScopedImageIntegrations"

	// ListeningEndpointsWithProcessesCap identifies the capability for sensor to process and send information about listening endpoints and their processes, AKA processes listening on ports
	ListeningEndpointsWithProcessesCap SensorCapability = "ListeningEndpointsWithProcesses"

	// DelegatedRegistryCap identifies the capability for a secured cluster to interact directly with registries (ie: for scanning images in local registries).
	DelegatedRegistryCap SensorCapability = "DelegatedRegistryCap"

	// SendDeduperStateOnReconnect identifies the capability to receive resource hashes from Central when reconnecting.
	SendDeduperStateOnReconnect = "SendDeduperStateOnReconnect"

	// ComplianceV2Integrations identifies the capability of central to support V2 integrations with compliance operator
	ComplianceV2Integrations = "ComplianceV2Integrations"

	// ScannerV4Supported identifies the capability of Central to support Scanner V4 related requests from Sensor.
	ScannerV4Supported = "ScannerV4Supported"

	// NetworkGraphInternalEntitiesSupported identifies the capability of Central (UI) to display internal entities in the network graph.
	NetworkGraphInternalEntitiesSupported = "NetworkGraphInternalEntitiesSupported"
)
