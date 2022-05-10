package env

var (
	// IncludeRBACInRisk toggles whether RBAC is included in the risk calculation.
	IncludeRBACInRisk = RegisterBooleanSetting("INCLUDE_RBAC_IN_RISK", true)
)
