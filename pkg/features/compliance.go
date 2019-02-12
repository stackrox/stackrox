package features

import "os"

type compliance struct{}

var (
	// Compliance is a feature flag for enabling compliance APIs
	Compliance = compliance{}
)

func (compliance) EnvVar() string {
	return "ROX_COMPLIANCE_ENABLED"
}

func (compliance) Name() string {
	return "Compliance"
}

func (c compliance) Enabled() bool {
	return isEnabled(os.Getenv(c.EnvVar()), true)
}
