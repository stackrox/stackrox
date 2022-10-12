package env

import "time"

var (
	// NodeScanInterval will set the duration for when to scan nodes for vulnerabilities (NodeScanV2)
	NodeScanInterval = registerDurationSetting("ROX_NODE_SCAN_INTERVAL", 4*time.Hour)
)
