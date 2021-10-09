package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

var (
	// csvExport enables CSV export of search results.
	csvExport = registerFeature("Enable CSV export of search results", "ROX_CSV_EXPORT", false)

	// ComplianceInRocksDB switches compliance over to using RocksDB instead of Bolt
	ComplianceInRocksDB = registerFeature("Switch compliance to using RocksDB", "ROX_COMPLIANCE_IN_ROCKSDB", true)

	// NetworkDetectionBaselineSimulation enables new features related to the baseline simulation part of the network detection experience.
	NetworkDetectionBaselineSimulation = registerFeature("Enable network detection baseline simulation", "ROX_NETWORK_DETECTION_BASELINE_SIMULATION", true)

	// NetworkDetectionBlockedFlows enables new features related to the blocked flows part of the network detection experience.
	NetworkDetectionBlockedFlows = registerFeature("Enable blocked network flows experience", "ROX_NETWORK_DETECTION_BLOCKED_FLOWS", false)

	// IntegrationsAsConfig enables loading integrations from config
	IntegrationsAsConfig = registerFeature("Enable loading integrations from config", "ROX_INTEGRATIONS_AS_CONFIG", false)

	// ScopedAccessControl enables scoped access control in core product
	ScopedAccessControl = registerFeature("Enable scoped access control in core product", "ROX_SCOPED_ACCESS_CONTROL_V2", true)

	// UpgradeRollback enables rollback to last central version after upgrade.
	UpgradeRollback = registerFeature("Enable rollback to last central version after upgrade", "ROX_ENABLE_ROLLBACK", true)

	// ComplianceOperatorCheckResults enables getting compliance results from the compliance operator
	ComplianceOperatorCheckResults = registerFeature("Enable fetching of compliance operator results", "ROX_COMPLIANCE_OPERATOR_INTEGRATION", true)

	// ActiveVulnManagement enables detection of active vulnerabilities
	ActiveVulnManagement = registerFeature("Enable detection of active vulnerabilities", "ROX_ACTIVE_VULN_MANAGEMENT", true)

	// AlternateProbeDownload enables alternate probe download solution for collector
	AlternateProbeDownload = registerFeature("Enable alternate probe download solution for collector", "ROX_COLLECTOR_ALT_PROBE_DOWNLOAD", false)

	PostgresPOC = registerFeature("Enable Postgres POC", "ROX_POSTGRES_POC", true)
)
