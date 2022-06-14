package tlsconfig

import "github.com/stackrox/rox/pkg/env"

// defaultAdditionalCAsDir is where the additional CAs are stored.
const defaultAdditionalCAsDir = "/usr/local/share/ca-certificates"

var (
	additionalCACertsDirPathSetting = env.RegisterSetting("ROX_MTLS_ADDITIONAL_CA_DIR", env.WithDefault(defaultAdditionalCAsDir))
)

// AdditionalCACertsDirPath returns the path where the additional CA certs are stored.
func AdditionalCACertsDirPath() string {
	return additionalCACertsDirPathSetting.Setting()
}
