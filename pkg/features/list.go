package features

var (
	// K8sRBAC is used to enable k8s rbac collection and processing
	// NB: When removing this feature flag, please also remove references to it in .circleci/config.yml
	K8sRBAC = registerFeature("Enable k8s RBAC objects collection and processing", "ROX_K8S_RBAC", true)

	// ScopedAccessControl controls whether scoped access control is enabled.
	// NB: When removing this feature flag, please also remove references to it in .circleci/config.yml
	ScopedAccessControl = registerFeature("Scoped Access Control", "ROX_SCOPED_ACCESS_CONTROL", true)

	// ClientCAAuth will enable authenticating to central via client certificate authentication
	ClientCAAuth = registerFeature("Client Certificate Authentication", "ROX_CLIENT_CA_AUTH", true)

	// PlaintextExposure enables specifying a port under which to expose the UI and API without TLS.
	PlaintextExposure = registerFeature("Plaintext UI & API exposure", "ROX_PLAINTEXT_EXPOSURE", true)

	// DBBackupRestoreV2 enables an improved database backup/restore experience.
	DBBackupRestoreV2 = registerFeature("Improved DB backup/restore experience", "ROX_DB_BACKUP_RESTORE_V2", true)

	// ScannerV2 enables scanner v2.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	ScannerV2 = registerFeature("Enable Scanner V2", "ROX_SCANNER_V2", true)

	// ConfigMgmtUI enables the config management UI.
	// NB: When removing this feature flag, remove references in ui/src/utils/featureFlags.js
	ConfigMgmtUI = registerFeature("Enable Config Mgmt UI", "ROX_CONFIG_MGMT_UI", false)

	// BadgerDB is used to enable BadgerDB as opposed to BoltDB for write heavy objects
	BadgerDB = registerFeature("Enable BadgerDB as opposed to BoltDB for write heavy objects", "ROX_BADGER_DB", false)

	// SensorAutoUpgrade controls the sensor autoupgrade feature.
	SensorAutoUpgrade = registerFeature("Enable Sensor Autoupgrades", "ROX_SENSOR_AUTOUPGRADE", false)
)
