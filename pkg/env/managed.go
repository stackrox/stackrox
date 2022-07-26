package env

var (
	// ManagedCentral is set to true to signal that the Central is Managed
	ManagedCentral = RegisterBooleanSetting("ROX_MANAGED_CENTRAL", false)
)
