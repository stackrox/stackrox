package chaos

import "github.com/stackrox/rox/pkg/env"

var (
	centralEndpointNoProxyEnv = env.RegisterSetting("ROX_CENTRAL_ENDPOINT_NO_PROXY")
	chaosProxyEnabledEnv      = env.RegisterBooleanSetting("ROX_CHAOS_PROXY_ENABLED", false)
	chaosProfileEnv           = env.RegisterSetting("ROX_CHAOS_PROFILE")
)

func chaosProfile() string {
	return chaosProfileEnv.Setting()
}

func originalCentralEndpoint() string {
	return centralEndpointNoProxyEnv.Setting()
}

// HasChaosProxy returns true if running with a chaos proxy between sensor and central.
func HasChaosProxy() bool {
	return chaosProxyEnabledEnv.BooleanSetting()
}
