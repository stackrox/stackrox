package env

var (
	// NetworkAccessLogEnv is the variable that clients can use to log HTTP requests to the central endpoint
	NetworkAccessLogEnv = RegisterBooleanSetting("ROX_NETWORK_ACCESS_LOG", false)
)

// LogNetworkRequest returns true if the network request should be logged
func LogNetworkRequest() bool {
	return NetworkAccessLogEnv.BooleanSetting()
}
