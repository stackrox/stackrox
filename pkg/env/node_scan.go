package env

import "time"

var (
	// NodeRescanInterval will set the duration for when to fetch node inventory to be scanned for vulnerabilities
	NodeRescanInterval = registerDurationSetting("ROX_NODE_RESCAN_INTERVAL", 4*time.Hour)
)
