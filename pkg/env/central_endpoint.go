package env

var (
	// CentralEndpoint is used to provide Central's reachable endpoint to other StackRox services.
	CentralEndpoint = RegisterSetting("ROX_CENTRAL_ENDPOINT", WithDefault("central.stackrox.svc:443"),
		StripAnyPrefix("https://", "http://"))
)
