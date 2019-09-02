package mtls

import "github.com/stackrox/rox/pkg/env"

var (
	caFilePathSetting    = env.RegisterSetting("ROX_MTLS_CA_FILE", env.WithDefault(defaultCACertFilePath))
	caKeyFilePathSetting = env.RegisterSetting("ROX_MTLS_CA_KEY_FILE", env.WithDefault(defaultCAKeyFilePath))
	certFilePathSetting  = env.RegisterSetting("ROX_MTLS_CERT_FILE", env.WithDefault(defaultCertFilePath))
	keyFilePathSetting   = env.RegisterSetting("ROX_MTLS_KEY_FILE", env.WithDefault(defaultKeyFilePath))
)

// CertFilePath returns the path where the certificate is stored.
func CertFilePath() string {
	return certFilePathSetting.Setting()
}

// KeyFilePath rteurns the path where the key is stored.
func KeyFilePath() string {
	return keyFilePathSetting.Setting()
}
