package env

import "time"

var (
	// RepoMappingUpdateMaxInitialWait is the maximum wait time before the first repository mapping data
	RepoMappingUpdateMaxInitialWait = registerDurationSetting("ROX_SCANNER_V4_CVSS_MAX_INITIAL_WAIT", 3*time.Minute)
)
