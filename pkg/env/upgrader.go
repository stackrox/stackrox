package env

var (
	// UpgraderCertsOnly is an environment variable constructed by the upgrader that instructs it to only
	// consider certs during operation.
	UpgraderCertsOnly = RegisterBooleanSetting("ROX_SENSOR_UPGRADER_CERTS_ONLY", false)
)
