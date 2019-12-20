package features

var (
	// ConfigMgmtUI enables the config management UI.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	ConfigMgmtUI = registerFeature("Enable Config Mgmt UI", "ROX_CONFIG_MGMT_UI", true)

	// VulnMgmtUI enables the vulnerability management UI.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	VulnMgmtUI = registerFeature("Enable Vulnerability Management UI", "ROX_VULN_MGMT_UI", false)

	// ManagedDB enabled the newly StackRox managed DB transaction sequencing.
	ManagedDB = registerFeature("Use managed sequencing for the embedded Badger DB", "ROX_MANAGED_DB", false)

	// Telemetry enables the telemetry features
	Telemetry = registerFeature("Enable support for telemetry", "ROX_TELEMETRY", false)
)
