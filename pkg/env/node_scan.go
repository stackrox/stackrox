package env

import "time"

var (
	// NodeScanningInterval is the base value of the interval duration between node scans.
	NodeScanningInterval = registerDurationSetting("ROX_NODE_SCANNING_INTERVAL", 4*time.Hour)

	// NodeScanningIntervalDeviation is the percentage by which each interval
	// duration between node scans will deviate from the base interval time. A value
	// equal or lower than zero disables deviation.
	NodeScanningIntervalDeviation = RegisterIntegerSetting("ROX_NODE_SCANNING_INTERVAL_DEVIATION", 10)

	// NodeScanningMaxInitialWait is the maximum wait time before the first node
	// scan, which is randomly generated. Set zero to disable the initial node
	// scanning wait time.
	NodeScanningMaxInitialWait = registerDurationSetting("ROX_NODE_SCANNING_MAX_INITIAL_WAIT", 5*time.Minute)
)
