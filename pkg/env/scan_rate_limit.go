package env

var (
	// ScanRateLimit sets the rate limit on the ImageScanInternal endpoint
	ScanRateLimit = RegisterIntegerSetting("ROX_SCAN_INTERNAL_RATE_LIMIT", 20)
)
