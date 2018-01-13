package features

import (
	"os"
)

const (
	// MTLSFlag is the environment variable controlling whether
	// MTLS verification is enabled.
	MTLSFlag = "ROX_FEATURE_MTLS"
)

type mtls struct{}

func (mtls) Name() string {
	return "Service-to-Service Mutual TLS"
}

func (mtls) EnvVar() string {
	return MTLSFlag
}

func (m mtls) Enabled() bool {
	return isEnabled(os.Getenv(m.EnvVar()), true)
}
