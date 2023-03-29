package env

var (
	// ActiveVulnMgmt defines if the active vuln mgmt feature is enabled
	ActiveVulnMgmt = RegisterBooleanSetting("ROX_ACTIVE_VULN_MGMT", false)
)
