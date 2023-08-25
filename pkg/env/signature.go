package env

var (
	// DisableSignatureFetching disables signature fetching within the reprocessing loop.
	DisableSignatureFetching = RegisterBooleanSetting("ROX_DISABLE_SIGNATURE_FETCHING", false)
)
