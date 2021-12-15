package env

import "time"

var (
	// ReprocessInterval will set the duration for which to reprocess all deployments and get new scans
	ReprocessInterval = registerDurationSetting("ROX_REPROCESSING_INTERVAL", 4*time.Hour)
	// ActiveVulnRefreshInterval will set the duration for which to refresh active components and vulnerabilities.
	ActiveVulnRefreshInterval = registerDurationSetting("ROX_ACTIVE_VULN_REFRESH_INTERVAL", 15*time.Minute)
	// VulnDeferralTimedReObserveInterval will set the duration for when to check to see if timed vuln deferrals need to be checked for expiry.
	VulnDeferralTimedReObserveInterval = registerDurationSetting("ROX_VULN_TIMED_DEFERRAL_REOBSERVE_INTERVAL", 1*time.Hour)
	// VulnDeferralFixableReObserveInterval will set the duration for when to check to see if "when fixable" vuln deferrals need to be checked for expiry.
	VulnDeferralFixableReObserveInterval = registerDurationSetting("ROX_VULN_FIXABLE_DEFERRAL_REOBSERVE_INTERVAL", 4*time.Hour)
)
