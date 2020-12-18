package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

var (
	// AdmissionControlService enables running admission control as a separate microservice.
	AdmissionControlService = registerFeature("Separate admission control microservice", "ROX_ADMISSION_CONTROL_SERVICE", true)

	// csvExport enables CSV export of search results.
	csvExport = registerFeature("Enable CSV export of search results", "ROX_CSV_EXPORT", false)

	// SupportSlimCollectorMode enables support for retrieving slim Collector bundles from central.
	SupportSlimCollectorMode = registerFeature("Support slim Collector mode", "ROX_SUPPORT_SLIM_COLLECTOR_MODE", true)

	// AwsSecurityHubIntegration enables the AWS Security Hub Integration UI.
	AwsSecurityHubIntegration = registerFeature("Show AWS Security Hub Integration in UI", "ROX_AWS_SECURITY_HUB_INTEGRATION", true)

	// NetworkGraphPorts enables port-related features in the network graph.
	NetworkGraphPorts = registerFeature("Enable port-related features in network graph", "ROX_NETWORK_GRAPH_PORTS", true)

	// NetworkFlowsSearchFilterUI enables client-side filtering for network flows.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	NetworkFlowsSearchFilterUI = registerFeature("Enable client-side network flows search", "ROX_NETWORK_FLOWS_SEARCH_FILTER_UI", true)

	// NetworkGraphExternalSrcs enables external sources feature in network graph.
	NetworkGraphExternalSrcs = registerFeature("Enable external sources feature in network graph ", "ROX_NETWORK_GRAPH_EXTERNAL_SRCS", true)

	// ComplianceInRocksDB switches compliance over to using RocksDB instead of Bolt
	ComplianceInRocksDB = registerFeature("Switch compliance to using RocksDB", "ROX_COMPLIANCE_IN_ROCKSDB", true)

	// CentralInstallationExperience enables new features related to the new installation experience.
	CentralInstallationExperience = registerFeature("Enable new installation user experience", "ROX_CENTRAL_INSTALLATION_EXPERIENCE", true)

	// SyslogIntegration enables UI for a Syslog integration in the Integrations section.
	SyslogIntegration = registerFeature("Enable UI for a Syslog integration in the Integrations section", "ROX_SYSLOG_INTEGRATION", true)

	// SensorInstallationExperience enables new features related to the new installation experience for sensor.
	SensorInstallationExperience = registerFeature("Enable new installation user experience for Sensor", "ROX_SENSOR_INSTALLATION_EXPERIENCE", false)

	// NetworkDetection enables new features related to the new network detection experience.
	NetworkDetection = registerFeature("Enable new network detection experience", "ROX_NETWORK_DETECTION", false)

	// HostScanning enables new features related to the new host scanning experience in VM.
	HostScanning = registerFeature("Enable new host scanning experience", "ROX_HOST_SCANNING", false)

	// DiscoveredAtImage enables discoveredAtImage graphQL field for CVE entity in VM.
	DiscoveredAtImage = registerFeature("Enable discoveredAtImage graphQL field for CVE entity", "ROX_DISCOVERED_AT_IMAGE", true)

	// SensorTLSChallenge enables Sensor to receive Centrals configured additional-ca an default certs.
	SensorTLSChallenge = registerFeature("Enable Sensor to receive default and additional CA certificates from Central", "ROX_SENSOR_TLS_CHALLENGE", true)

	// K8sAuditLogDetection enables Sensor to collect k8s audit logs using cloud integrations and generate policy violations
	// based on k8s audit log events
	K8sAuditLogDetection = registerFeature("Enable Sensor to collect k8s audit logs in EKS and GKE clusters and generate violations based on audit log events", "ROX_K8S_AUDIT_LOG_DETECTION", false)

	// IntegrationsAsConfig enables loading integrations from config
	IntegrationsAsConfig = registerFeature("Enable loading integrations from config", "ROX_INTEGRATIONS_AS_CONFIG", false)
)
