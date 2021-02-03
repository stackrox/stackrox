package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

var (
	// AdmissionControlService enables running admission control as a separate microservice.
	AdmissionControlService = registerFeature("Separate admission control microservice", "ROX_ADMISSION_CONTROL_SERVICE", true)

	// csvExport enables CSV export of search results.
	csvExport = registerFeature("Enable CSV export of search results", "ROX_CSV_EXPORT", false)

	// SupportSlimCollectorMode enables support for retrieving slim Collector bundles from central.
	SupportSlimCollectorMode = registerFeature("Support slim Collector mode", "ROX_SUPPORT_SLIM_COLLECTOR_MODE", true)

	// ComplianceInRocksDB switches compliance over to using RocksDB instead of Bolt
	ComplianceInRocksDB = registerFeature("Switch compliance to using RocksDB", "ROX_COMPLIANCE_IN_ROCKSDB", true)

	// SensorInstallationExperience enables new features related to the new installation experience for sensor.
	SensorInstallationExperience = registerFeature("Enable new installation user experience for Sensor", "ROX_SENSOR_INSTALLATION_EXPERIENCE", true)

	// NetworkDetection enables new features related to the new network detection experience.
	NetworkDetection = registerFeature("Enable new network detection experience", "ROX_NETWORK_DETECTION", true)

	// NetworkDetectionBaselineViolation enables new features related to the baseline violation part of the network detection experience.
	NetworkDetectionBaselineViolation = registerFeature("Enable network detection baseline violation", "ROX_NETWORK_DETECTION_BASELINE_VIOLATION", false)

	// HostScanning enables new features related to the new host scanning experience in VM.
	HostScanning = registerFeature("Enable new host scanning experience", "ROX_HOST_SCANNING", true)

	// SensorTLSChallenge enables Sensor to receive Centrals configured additional-ca an default certs.
	SensorTLSChallenge = registerFeature("Enable Sensor to receive default and additional CA certificates from Central", "ROX_SENSOR_TLS_CHALLENGE", true)

	// K8sEventDetection enables detection of kubernetes events.
	K8sEventDetection = registerFeature("Enable detection of kubernetes events", "ROX_K8S_EVENTS_DETECTION", true)

	// IntegrationsAsConfig enables loading integrations from config
	IntegrationsAsConfig = registerFeature("Enable loading integrations from config", "ROX_INTEGRATIONS_AS_CONFIG", false)

	// ScopedAccessControl enables scoped access control in core product
	ScopedAccessControl = registerFeature("Enable scoped access control in core product", "ROX_SCOPED_ACCESS_CONTROL_V2", false)

	// K8sAuditLogDetection enables detection of kubernetes audit log based event detection.
	K8sAuditLogDetection = registerFeature("Enable detection of kubernetes audit log based event detection", "ROX_K8S_AUDIT_LOG_DETECTION", false)
)
