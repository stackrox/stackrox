package env

import "time"

var (
	// CvssDataUpdateMaxInitialWait is the maximum wait time before the first CVSS data
	CvssDataUpdateMaxInitialWait = registerDurationSetting("ROX_SCANNER_V4_CVSS_MAX_INITIAL_WAIT", 3*time.Minute)
)
