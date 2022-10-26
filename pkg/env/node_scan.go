package env

import "time"

var (
	NodeRescanIntervalDefault = 4 * time.Hour
	// NodeRescanInterval will set the duration for when to scan nodes for vulnerabilities (NodeScanV2)
	NodeRescanInterval = registerDurationSetting("ROX_NODE_RESCAN_INTERVAL", NodeRescanIntervalDefault)
)

// GetNodeRescanInterval returns NodeRescanInterval if positive, otherwise returns the default
func GetNodeRescanInterval() time.Duration {
	if NodeRescanInterval.DurationSetting() <= 0 {
		log.Warnf("Negative or zero duration found. Setting to %s.", NodeRescanIntervalDefault.String())
		return NodeRescanIntervalDefault
	}
	return NodeRescanInterval.DurationSetting()
}
