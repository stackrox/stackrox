package env

import "time"

var (
	// NodeRescanInterval will set the duration for when to scan nodes for vulnerabilities (NodeScanV2)
	NodeRescanInterval = registerDurationSetting("ROX_NODE_RESCAN_INTERVAL", 4*time.Hour)
)
