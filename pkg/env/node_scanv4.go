package env

var (
	// NodeScanningV4HostPath sets the path where the R/O host node filesystem is mounted
	// that should be scanned
	NodeScanningV4HostPath = RegisterSetting("ROX_NODE_SCANNING_V4_HOST_PATH", WithDefault("/host"))
)
