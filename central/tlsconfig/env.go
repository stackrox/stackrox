package tlsconfig

import "github.com/stackrox/rox/pkg/env"

const (
	// defaultAdditionalCAsDir is where the additional CAs are stored.
	defaultAdditionalCAsDir = "/usr/local/share/ca-certificates"
	// MTLSAdditionalCADirEnvName is the env var name for the additionalCA directory.
	MTLSAdditionalCADirEnvName = "ROX_MTLS_ADDITIONAL_CA_DIR"
)

var (
	additionalCACertsDirPathSetting = env.RegisterSetting(MTLSAdditionalCADirEnvName, env.WithDefault(defaultAdditionalCAsDir))
)

// AdditionalCACertsDirPath returns the path where the additional CA certs are stored.
func AdditionalCACertsDirPath() string {
	return additionalCACertsDirPathSetting.Setting()
}
