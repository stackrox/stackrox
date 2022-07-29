package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

var (
	// csvExport enables CSV export of search results.
	csvExport = registerFeature("Enable CSV export of search results", "ROX_CSV_EXPORT", false)

	// NetworkDetectionBaselineSimulation enables new features related to the baseline simulation part of the network detection experience.
	NetworkDetectionBaselineSimulation = registerFeature("Enable network detection baseline simulation", "ROX_NETWORK_DETECTION_BASELINE_SIMULATION", true)

	// IntegrationsAsConfig enables loading integrations from config
	IntegrationsAsConfig = registerFeature("Enable loading integrations from config", "ROX_INTEGRATIONS_AS_CONFIG", false)

	// UpgradeRollback enables rollback to last central version after upgrade.
	UpgradeRollback = registerFeature("Enable rollback to last central version after upgrade", "ROX_ENABLE_ROLLBACK", true)

	// ComplianceOperatorCheckResults enables getting compliance results from the compliance operator
	ComplianceOperatorCheckResults = registerFeature("Enable fetching of compliance operator results", "ROX_COMPLIANCE_OPERATOR_INTEGRATION", true)

	// SystemHealthPatternFly enables the Pattern Fly version of System Health page. (used in the front-end app only)
	SystemHealthPatternFly = registerFeature("Enable Pattern Fly version of System Health page", "ROX_SYSTEM_HEALTH_PF", false)

	// PoliciesPatternFly enables the PatternFly version of Policies. (used in the front-end app only)
	PoliciesPatternFly = registerFeature("Enable PatternFly version of Policies", "ROX_POLICIES_PATTERNFLY", true)

	// LocalImageScanning enables OpenShift local-image scanning.
	LocalImageScanning = registerFeature("Enable OpenShift local-image scanning", "ROX_LOCAL_IMAGE_SCANNING", true)

	// PostgresDatastore enables Postgres datastore.
	PostgresDatastore = registerFeature("Enable Postgres Datastore", "ROX_POSTGRES_DATASTORE", false)

	// FrontendVMUpdates enables Frontend VM Updates.
	FrontendVMUpdates = registerFeature("Enable Frontend VM Updates", "ROX_FRONTEND_VM_UPDATES", false)

	// ECRAutoIntegration enables detection of ECR-based deployments to generate auto-integrations from ECR auth tokens.
	ECRAutoIntegration = registerFeature("Enable ECR auto-integrations when running on AWS nodes", "ROX_ECR_AUTO_INTEGRATION", true)

	// NetworkPolicySystemPolicy enables two system policy fields (Missing (Ingress|Egress) Network Policy) to check deployments
	// against network policies applied in the secured cluster.
	NetworkPolicySystemPolicy = registerFeature("Enable NetworkPolicy-related system policy fields", "ROX_NETPOL_FIELDS", true)

	// NewPolicyCategories enables new policy categories as first-class entities.
	NewPolicyCategories = registerFeature("Enable new policy categories as first-class entities", "ROX_NEW_POLICY_CATEGORIES", false)

	// SecurityMetricsPhaseOne enables the PatternFly version of the main dashboard with Action Widgets. (used in the front-end app only)
	SecurityMetricsPhaseOne = registerFeature("Enable PatternFly version of Security Metrics Dashboard", "ROX_SECURITY_METRICS_PHASE_ONE", true)

	// DecommissionedClusterRetention enables the setting in System Configuration.
	DecommissionedClusterRetention = registerFeature("Enable Decommissioned Cluster Retention in System Configuration", "ROX_DECOMMISSIONED_CLUSTER_RETENTION", true)
)
