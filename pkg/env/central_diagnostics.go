package env

var (
	// EnableCentralDiagnostics is set to true to signal that diagnostic bundles or dumps should contain information about central or the environment that central runs in.
	EnableCentralDiagnostics = RegisterBooleanSetting("ROX_ENABLE_CENTRAL_DIAGNOSTICS", true)

	// EnableCentralDatabaseDiagnostics is set to true to signal that diagnostic bundles or dumps should contain information about central database or the environment that central runs in.
	EnableCentralDatabaseDiagnostics = RegisterBooleanSetting("ROX_ENABLE_CENTRAL_DATABASE_DIAGNOSTICS", true)
)
