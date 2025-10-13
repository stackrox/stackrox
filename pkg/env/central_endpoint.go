package env

// These environment variables are used in deployment files.
// Please check the files before deleting and check their usage in the code before editing.
var (
	// CentralEndpoint is used to provide Central's reachable endpoint to other StackRox services.
	CentralEndpoint = RegisterSetting("ROX_CENTRAL_ENDPOINT", WithDefault("central.stackrox.svc:443"),
		StripAnyPrefix("https://", "http://"))
)
