package env

var (
	// ExternalIPsPruning enables the pruning of 'discovered' external entities.
	// The pruning is always enabled when ROX_EXTERNAL_IPS is enabled.
	ExternalIPsPruning = RegisterBooleanSetting("ROX_EXTERNAL_IPS_PRUNING", false)
)
