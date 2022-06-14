package mtls

import "github.com/stackrox/stackrox/pkg/env"

const (
	// CAFileEnvName is the env variable name for the CA file
	CAFileEnvName = "ROX_MTLS_CA_FILE"
	// CAKeyFileEnvName is the env variable name for the CA key file
	CAKeyFileEnvName = "ROX_MTLS_CA_KEY_FILE"
	// CertFilePathEnvName is the env variable name for the cert file
	CertFilePathEnvName = "ROX_MTLS_CERT_FILE"
	// KeyFileEnvName is the env variable name for the key file which signed the cert
	KeyFileEnvName = "ROX_MTLS_KEY_FILE"
)

var (
	caFilePathSetting    = env.RegisterSetting(CAFileEnvName, env.WithDefault(defaultCACertFilePath))
	caKeyFilePathSetting = env.RegisterSetting(CAKeyFileEnvName, env.WithDefault(defaultCAKeyFilePath))
	certFilePathSetting  = env.RegisterSetting(CertFilePathEnvName, env.WithDefault(defaultCertFilePath))
	keyFilePathSetting   = env.RegisterSetting(KeyFileEnvName, env.WithDefault(defaultKeyFilePath))
)

// CAFilePath returns the path where the CA certificate is stored.
func CAFilePath() string {
	return caFilePathSetting.Setting()
}

// CertFilePath returns the path where the certificate is stored.
func CertFilePath() string {
	return certFilePathSetting.Setting()
}

// KeyFilePath returns the path where the key is stored.
func KeyFilePath() string {
	return keyFilePathSetting.Setting()
}
