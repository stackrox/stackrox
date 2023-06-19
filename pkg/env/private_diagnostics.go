package env

const defaultPrivatePortSetting = ":9095"

var (
	// PrivateDiagnosticsEnabled toggles whether diagnostic handlers are available on private endpoint.
	PrivateDiagnosticsEnabled = RegisterBooleanSetting("ROX_PRIVATE_DIAGNOSTICS", false)

	// PrivatePortSetting has the :port or host:port string for listening for private endpoints server.
	PrivatePortSetting = RegisterSetting("ROX_PRIVATE_PORT", WithDefault(defaultPrivatePortSetting))
)
