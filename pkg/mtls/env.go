package mtls

import "github.com/stackrox/rox/pkg/env"

const (
	// CAFileEnvName is the env variable name for the CA file
	CAFileEnvName = "ROX_MTLS_CA_FILE"
	// CAKeyFileEnvName is the env variable name for the CA key file
	CAKeyFileEnvName = "ROX_MTLS_CA_KEY_FILE"
	// CertFilePathEnvName is the env variable name for the cert file
	CertFilePathEnvName = "ROX_MTLS_CERT_FILE"
	// KeyFileEnvName is the env variable name for the key file which signed the cert
	KeyFileEnvName = "ROX_MTLS_KEY_FILE"

	// CentralDBCertFilePathEnvName is the env variable name for the central-db cert file
	CentralDBCertFilePathEnvName = "ROX_MTLS_CENTRAL_DB_CERT_FILE"
	// CentralDBKeyFileEnvName is the env variable name for the key file which signed the central-db cert
	CentralDBKeyFileEnvName = "ROX_MTLS_CENTRAL_DB_KEY_FILE"
)

var (
	caFilePathSetting    = env.RegisterSetting(CAFileEnvName, env.WithDefault(defaultCACertFilePath))
	caKeyFilePathSetting = env.RegisterSetting(CAKeyFileEnvName, env.WithDefault(defaultCAKeyFilePath))
	certFilePathSetting  = env.RegisterSetting(CertFilePathEnvName, env.WithDefault(defaultCertFilePath))
	keyFilePathSetting   = env.RegisterSetting(KeyFileEnvName, env.WithDefault(defaultKeyFilePath))

	centraldbCertFilePathSetting = env.RegisterSetting(CentralDBCertFilePathEnvName, env.WithDefault(defaultCentralDBCertFilePath))
	centraldbKeyFilePathSetting  = env.RegisterSetting(CentralDBKeyFileEnvName, env.WithDefault(defaultCentralDBKeyFilePath))
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

// CentralDBCertFilePath returns the path where the central-db certificate is stored
func CentralDBCertFilePath() string {
	return centraldbCertFilePathSetting.Setting()
}

// CentralDBKeyFilePath returns the path where the key for central-db certificate is stored
func CentralDBKeyFilePath() string {
	return centraldbKeyFilePathSetting.Setting()
}
