package chaos

import (
	"strings"

	"github.com/stackrox/rox/pkg/env"
)

var (
	centralEndpointNoProxyEnv = env.RegisterSetting("ROX_CENTRAL_ENDPOINT_NO_PROXY")
	chaosProfileEnv           = env.RegisterSetting("ROX_CHAOS_PROFILE")
)

func chaosProfile() string {
	return chaosProfileEnv.Setting()
}

func originalCentralEndpoint() string {
	value := centralEndpointNoProxyEnv.Setting()
	return strings.TrimSpace(value)
}

// HasChaosProxy returns true if running with a chaos proxy between sensor and central.
func HasChaosProxy() bool {
	return chaosProfile() != ""
}
