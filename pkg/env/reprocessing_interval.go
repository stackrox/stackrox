package env

import "time"

var (
	// ReprocessInterval will set the duration for which to reprocess all deployments and get new scans
	ReprocessInterval = registerDurationSetting("ROX_REPROCESSING_INTERVAL", 4*time.Hour)
	// ActiveVulnRefreshInterval will set the duration for which to refresh active components and vulnerabilities.
	ActiveVulnRefreshInterval = registerDurationSetting("ROX_ACTIVE_VULN_REFRESH_INTERVAL", 15*time.Minute)
)
