package env

import "time"

var (
	// ScanTimeout defines the image scan timeout duration.
	ScanTimeout = registerDurationSetting("ROX_SCAN_TIMEOUT", 10*time.Minute)

	// SBOMGenerationMaxReqSizeBytes defines the maximum allowed size of an SBOM generation API request.
	SBOMGenerationMaxReqSizeBytes = RegisterIntegerSetting("ROX_SBOM_GEN_MAX_REQ_SIZE_BYTES", 100*1024)

	// SBOMScanMaxReqSizeBytes defines the maximum allowed size of an SBOM scan API request (100 MB).
	SBOMScanMaxReqSizeBytes = RegisterIntegerSetting("ROX_SBOM_SCAN_MAX_REQ_SIZE_BYTES", 100*1024*1024)
)
