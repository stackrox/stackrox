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

	// UpgradeRollback enables rollback to last central version after upgrade.
	UpgradeRollback = registerFeature("Enable rollback to last central version after upgrade", "ROX_ENABLE_ROLLBACK", true)

	// ComplianceOperatorCheckResults enables getting compliance results from the compliance operator
	ComplianceOperatorCheckResults = registerFeature("Enable fetching of compliance operator results", "ROX_COMPLIANCE_OPERATOR_INTEGRATION", true)

	// ActiveVulnManagement enables detection of active vulnerabilities
	ActiveVulnManagement = registerFeature("Enable detection of active vulnerabilities", "ROX_ACTIVE_VULN_MANAGEMENT", true)

	// SystemHealthPatternFly enables the Pattern Fly version of System Health page. (used in the front-end app only)
	SystemHealthPatternFly = registerFeature("Enable Pattern Fly version of System Health page", "ROX_SYSTEM_HEALTH_PF", false)

	// PoliciesPatternFly enables the PatternFly version of Policies. (used in the front-end app only)
	PoliciesPatternFly = registerFeature("Enable PatternFly version of Policies", "ROX_POLICIES_PATTERNFLY", true)

	// VulnReporting enables scheduled vulnerability reporting workflow, that allows the creation and management of vulnerability reporting configurations.
	VulnReporting = registerFeature("Enable creation of scheduled vulnerability reports to be sent via email notifier", "ROX_VULN_REPORTING", true)

	// LocalImageScanning enables OpenShift local-image scanning.
	LocalImageScanning = registerFeature("Enable OpenShift local-image scanning", "ROX_LOCAL_IMAGE_SCANNING", true)

	// ImageSignatureVerification enables image signature verification.
	ImageSignatureVerification = registerFeature("Enable Image Signature Verification workflow", "ROX_VERIFY_IMAGE_SIGNATURE", false)

	// PostgresDatastore enables Postgres datastore.
	PostgresDatastore = registerFeature("Enable Postgres Datastore", "ROX_POSTGRES_DATASTORE", false)

	// ECRAutoIntegration enables detection of ECR-based deployments to generate auto-integrations from ECR auth tokens.
	ECRAutoIntegration = registerFeature("Enable ECR auto-integrations when running on AWS nodes", "ROX_ECR_AUTO_INTEGRATION", true)

	// NetworkPolicySystemPolicy enables two system policy fields (Missing (Ingress|Egress) Network Policy) to check deployments
	// against network policies applied in the secured cluster.
	NetworkPolicySystemPolicy = registerFeature("Enable NetworkPolicy-related system policy fields", "ROX_NETPOL_FIELDS", false)
)
