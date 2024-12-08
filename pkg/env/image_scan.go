package env

import "time"

var (
	// ScanTimeout defines the image scan timeout duration.
	ScanTimeout = registerDurationSetting("ROX_SCAN_TIMEOUT", 10*time.Minute)

	// ResetClusterLocalOnCentralScan when true Central will be able to reprocess images formerly
	// flagged as cluster local (indexed via a secured cluster). When Central receives a scan request
	// for an image previously flagged as cluster local and the scan has expired, the cluster local flag
	// will be reset (to false) and a full scan via Central attempted. On success the image will be
	// saved to the DB allowing future Central reprocessing runs to scan the image (vs. skip).
	//
	// TODO(ROX-26341): Remove this when we have confidence it is behaving as desired.
	ResetClusterLocalOnCentralScan = RegisterBooleanSetting("ROX_RESET_CLUSTER_LOCAL_ON_CENTRAL_SCAN", true)
)
