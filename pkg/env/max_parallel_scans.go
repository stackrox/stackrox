package env

var (
	// MaxParallelImageScanInternal sets the max number of parallel scans collectively on the ScanImageInternal, EnrichLocalImageInternal,
	// and GetImageVulnerabilitiesInternal endpoints and separately sets the max active local scans in a secured cluster
	MaxParallelImageScanInternal = RegisterIntegerSetting("ROX_MAX_PARALLEL_IMAGE_SCAN_INTERNAL", 30)

	// MaxParallelDelegatedScanInternal defines the maximum number of parallel delegated scans initiated from Central.
	// Since it consumes a portion of the total maximum parallel scans, it must be set lower than the collective limit
	// for ScanImageInternal, EnrichLocalImageInternal, and GetImageVulnerabilitiesInternal endpoints.
	// It separately configures the maximum number of active local scans within a secured cluster.
	MaxParallelDelegatedScanInternal = RegisterIntegerSetting("ROX_MAX_PARALLEL_DELEGATED_SCAN_INTERNAL", 5)
)
