package env

var (
	// MaxParallelImageScanInternal sets the max number of parallel scans collectively on the ScanImageInternal, EnrichLocalImageInternal,
	// and GetImageVulnerabilitiesInternal endpoints and separately sets the max active local scans in a secured cluster
	MaxParallelImageScanInternal = RegisterIntegerSetting("ROX_MAX_PARALLEL_IMAGE_SCAN_INTERNAL", 30)

	// MaxParallelAdHocScan defines the maximum number of parallel ad-hoc roxctl delegated scans initiated from Central.
	// It must be set lower than the collective limit for ScanImageInternal, EnrichLocalImageInternal, and GetImageVulnerabilitiesInternal endpoints.
	MaxParallelAdHocScan = RegisterIntegerSetting("ROX_MAX_PARALLEL_ADHOC_SCAN_INTERNAL", 5)
)
