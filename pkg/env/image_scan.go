package env

import "time"

var (
	// ScanTimeout defines the image scan timeout duration.
	ScanTimeout = registerDurationSetting("ROX_SCAN_TIMEOUT", 6*time.Minute)
)
