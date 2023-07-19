package env

var (
	// RegistryMirroringEnabled enables processing registry mirrors during image enrichment.
	RegistryMirroringEnabled = RegisterBooleanSetting("ROX_REGISTRY_MIRRORING_ENABLED", false)
)
