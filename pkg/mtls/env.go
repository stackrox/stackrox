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
	// CrsFilePathEnvName is the env variable name for the CRS file.
	CrsFilePathEnvName = "ROX_MTLS_CRS_FILE"
)

var (
	// CAFilePathSetting allows configuring the CA certificate from the environment.
	CAFilePathSetting = env.RegisterSetting(CAFileEnvName, env.WithDefault(defaultCACertFilePath))
	// CAKeyFilePathSetting allows configuring the CA key from the environment.
	CAKeyFilePathSetting = env.RegisterSetting(CAKeyFileEnvName, env.WithDefault(defaultCAKeyFilePath))
	// CertFilePathSetting allows configuring the MTLS certificate from the environment.
	CertFilePathSetting = env.RegisterSetting(CertFilePathEnvName, env.WithDefault(defaultCertFilePath))
	// KeyFilePathSetting allows configuring the MTLS key from the environment.
	KeyFilePathSetting = env.RegisterSetting(KeyFileEnvName, env.WithDefault(defaultKeyFilePath))
	// CrsFilePathSetting allows configuring the CRS from the environment.
	CrsFilePathSetting = env.RegisterSetting(CrsFilePathEnvName, env.WithDefault(defaultCrsFilePath))
)

// CAFilePath returns the path where the CA certificate is stored.
func CAFilePath() string {
	return CAFilePathSetting.Setting()
}

// CertFilePath returns the path where the certificate is stored.
func CertFilePath() string {
	return CertFilePathSetting.Setting()
}

// KeyFilePath returns the path where the key is stored.
func KeyFilePath() string {
	return KeyFilePathSetting.Setting()
}

// CrsFilePath returns the path where the CRS certificate is stored.
func CrsFilePath() string {
	return CrsFilePathSetting.Setting()
}
