package env

var (
	// MaxParallelImageScanInternal sets the max number of parallel scans collectively on the ScanImageInternal, EnrichLocalImageInternal,
	// and GetImageVulnerabilitiesInternal endpoints and separately sets the max active local scans in a secured cluster
	MaxParallelImageScanInternal = RegisterIntegerSetting("ROX_MAX_PARALLEL_IMAGE_SCAN_INTERNAL", 30)
)
