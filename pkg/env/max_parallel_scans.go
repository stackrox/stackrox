package env

var (
	// MaxParallelImageScanInternal sets the max number of parallel scans collectively on the ScanImageInternal, EnrichLocalImageInternal,
	// and GetImageVulnerabilitiesInternal endpoints and separately sets the max active local scans in a secured cluster
	MaxParallelImageScanInternal = RegisterIntegerSetting("ROX_MAX_PARALLEL_IMAGE_SCAN_INTERNAL", 30)

	// MaxParallelAdHocScan defines the maximum number of parallel ad hoc delegated scans initiated from Central.
	// The value is subtracted from MaxParallelImageScanInternal. To ensure that ad hoc requests can always
	// be handled it will be set to 1 less than MaxParallelImageScanInternal if it exceeds MaxParallelImageScanInternal.
	MaxParallelAdHocScan = RegisterIntegerSetting("ROX_MAX_PARALLEL_AD_HOC_SCAN_INTERNAL", 5)
)
