package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

var (
	// csvExport enables CSV export of search results.
	csvExport = registerFeature("Enable CSV export of search results", "ROX_CSV_EXPORT", false)

	// ComplianceInRocksDB switches compliance over to using RocksDB instead of Bolt
	ComplianceInRocksDB = registerFeature("Switch compliance to using RocksDB", "ROX_COMPLIANCE_IN_ROCKSDB", true)

	// SensorInstallationExperience enables new features related to the new installation experience for sensor.
	SensorInstallationExperience = registerFeature("Enable new installation user experience for Sensor", "ROX_SENSOR_INSTALLATION_EXPERIENCE", true)

	// NetworkDetectionBaselineViolation enables new features related to the baseline violation part of the network detection experience.
	NetworkDetectionBaselineViolation = registerFeature("Enable network detection baseline violation", "ROX_NETWORK_DETECTION_BASELINE_VIOLATION", true)

	// NetworkDetectionBaselineSimulation enables new features related to the baseline simulation part of the network detection experience.
	NetworkDetectionBaselineSimulation = registerFeature("Enable network detection baseline simulation", "ROX_NETWORK_DETECTION_BASELINE_SIMULATION", false)

	// NetworkDetectionBlockedFlows enables new features related to the blocked flows part of the network detection experience.
	NetworkDetectionBlockedFlows = registerFeature("Enable blocked network flows experience", "ROX_NETWORK_DETECTION_BLOCKED_FLOWS", false)

	// IntegrationsAsConfig enables loading integrations from config
	IntegrationsAsConfig = registerFeature("Enable loading integrations from config", "ROX_INTEGRATIONS_AS_CONFIG", false)

	// ScopedAccessControl enables scoped access control in core product
	ScopedAccessControl = registerFeature("Enable scoped access control in core product", "ROX_SCOPED_ACCESS_CONTROL_V2", true)

	// K8sAuditLogDetection enables detection of kubernetes audit log based event detection.
	K8sAuditLogDetection = registerFeature("Enable detection of kubernetes audit log based event detection", "ROX_K8S_AUDIT_LOG_DETECTION", true)

	// InactiveImageScanningUI enables UI to facilitate scanning of inactive images.
	InactiveImageScanningUI = registerFeature("Enable UI to facilitate scanning of inactive images", "ROX_INACTIVE_IMAGE_SCANNING_UI", true)

	// UpgradeRollback enables rollback to last central version after upgrade.
	UpgradeRollback = registerFeature("Enable rollback to last central version after upgrade", "ROX_ENABLE_ROLLBACK", true)

	// ComplianceOperatorCheckResults enables getting compliance results from the compliance operator
	ComplianceOperatorCheckResults = registerFeature("Enable fetching of compliance operator results", "ROX_COMPLIANCE_OPERATOR_INTEGRATION", true)
)
