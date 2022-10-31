package env

import "time"

var (
	nodeRescanIntervalDefault = 4 * time.Hour
	// nodeRescanInterval will set the duration for when to scan nodes for vulnerabilities (NodeScanV2)
	nodeRescanInterval = registerDurationSetting("ROX_NODE_RESCAN_INTERVAL", nodeRescanIntervalDefault)
)

// GetNodeRescanInterval returns NodeRescanInterval if positive, otherwise returns the default
func GetNodeRescanInterval() time.Duration {
	if nodeRescanInterval.DurationSetting() <= 0 {
		log.Warnf("Negative or zero duration found. Setting to %s.", nodeRescanIntervalDefault.String())
		return nodeRescanIntervalDefault
	}
	return nodeRescanInterval.DurationSetting()
}
