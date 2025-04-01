package env

var (
	// MaxParallelImageScanInternal sets the max number of parallel scans collectively on the ScanImageInternal, EnrichLocalImageInternal,
	// and GetImageVulnerabilitiesInternal endpoints and separately sets the max active local scans in a secured cluster
	MaxParallelImageScanInternal = RegisterIntegerSetting("ROX_MAX_PARALLEL_IMAGE_SCAN_INTERNAL", 30)

	// MaxParallelAdHocScan defines the maximum number of parallel ad-hoc roxctl delegated scans initiated from Central.
	// The value is subtracted from MaxParallelImageScanInternal and must be less than MaxParallelImageScanInternal.
	// This ensures that ad-hoc requests can always be handled.
	// If this value exceeds MaxParallelImageScanInternal, MaxParallelImageScanInternal will be set to 10 higher than this value to prevent scan failures.
	MaxParallelAdHocScan = RegisterIntegerSetting("ROX_MAX_PARALLEL_AD_HOC_SCAN_INTERNAL", 5)
)
