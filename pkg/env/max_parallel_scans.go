package env

var (
	// MaxParallelImageScanInternal sets the max number of parallel scans on the ImageScanInternal endpoint.
	MaxParallelImageScanInternal = RegisterIntegerSetting("ROX_MAX_PARALLEL_IMAGE_SCAN_INTERNAL", 30)
)
