package features

var (
	// ConfigMgmtUI enables the config management UI.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	ConfigMgmtUI = registerFeature("Enable Config Mgmt UI", "ROX_CONFIG_MGMT_UI", true)

	// BadgerDB is used to enable BadgerDB as opposed to BoltDB for write heavy objects
	BadgerDB = registerFeature("Enable BadgerDB as opposed to BoltDB for write heavy objects", "ROX_BADGER_DB", true)

	// VulnMgmtUI enables the vulnerability management UI.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	VulnMgmtUI = registerFeature("Enable Vulnerability Management UI", "ROX_VULN_MGMT_UI", false)

	// ProbeUpload enables the possibility to upload collector probes to central.
	ProbeUpload = registerFeature("Enable support for uploading collector probes", "ROX_PROBE_UPLOAD", true)

	// LanguageScanner enables the deployment of the language scanner
	LanguageScanner = registerFeature("Enable support for the version of the image scanner that detects on languages", "ROX_LANGUAGE_SCANNER", false)
)
