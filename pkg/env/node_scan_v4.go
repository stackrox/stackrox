package env

var (
	// NodeScanningV4HostPath sets the path where the R/O host node filesystem is mounted to the container
	// that should be scanned by Scanners NodeIndexer
	NodeScanningV4HostPath = RegisterSetting("ROX_NODE_SCANNING_V4_HOST_PATH", WithDefault("/host"))

	// NodeScanningV4Enabled defines whether Compliance will actually run scanning code
	NodeScanningV4Enabled = RegisterBooleanSetting("ROX_NODE_SCANNING_V4_ENABLED", true) // FIXME: Default to false
)
