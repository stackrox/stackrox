package features

import "os"

const (
	runtimePoliciesEnvVar      = "ROX_RUNTIME_POLICIES"
	runtimePoliciesFeatureName = "Runtime Policies"
)

type runtimePolicies struct{}

func (r runtimePolicies) Name() string {
	return runtimePoliciesFeatureName
}

func (r runtimePolicies) EnvVar() string {
	return runtimePoliciesEnvVar
}

func (r runtimePolicies) Enabled() bool {
	return isEnabled(os.Getenv(runtimePoliciesEnvVar), false)
}
