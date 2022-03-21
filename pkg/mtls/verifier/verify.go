package verifier

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/stackrox/rox/pkg/mtls"
)

// A TLSConfigurer instantiates the appropriate TLS config for your environment.
type TLSConfigurer interface {
	TLSConfig() (*tls.Config, error)
}

// TLSConfigurerFunc wraps a plain function as a TLSConfigurer.
type TLSConfigurerFunc func() (*tls.Config, error)

// TLSConfig returns the TLS config by invoking f.
func (f TLSConfigurerFunc) TLSConfig() (*tls.Config, error) {
	return f()
}

// TrustedCertPool creates a CertPool that contains the CA certificate.
func TrustedCertPool() (*x509.CertPool, error) {
	caCert, _, err := mtls.CACert()
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	certPool.AddCert(caCert)
	return certPool, nil
}

// SystemCertPool returns all systems CAs including application specific CA
func SystemCertPool() (*x509.CertPool, error) {
	caCert, _, err := mtls.CACert()
	if err != nil {
		return nil, err
	}
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	certPool.AddCert(caCert)
	return certPool, nil
}

// DefaultTLSServerConfig returns the default TLS config for servers in StackRox
func DefaultTLSServerConfig() *tls.Config {
	// Government clients require TLS >=1.2 and require that AES-256 be preferred over AES-128
	return &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		ClientAuth: tls.VerifyClientCertIfGiven,
		NextProtos: []string{"h2"},
	}
}
