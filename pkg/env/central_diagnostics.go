package env

var (
	// DisableCentralDiagnostics is set to true to signal that diagnostic bundles or dumps should not contain any information about central or the environment that central runs in.
	DisableCentralDiagnostics = RegisterBooleanSetting("ROX_DISABLE_CENTRAL_DIAGNOSTICS", false)
)
