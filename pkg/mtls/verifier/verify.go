package verifier

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/mtls"
)

// A TLSConfigurer instantiates the appropriate TLS config for your environment.
//
//go:generate mockgen-wrapper
type TLSConfigurer interface {
	TLSConfig() (*tls.Config, error)
}

// TLSConfigurerFunc wraps a plain function as a TLSConfigurer.
type TLSConfigurerFunc func() (*tls.Config, error)

// TLSConfig returns the TLS config by invoking f.
func (f TLSConfigurerFunc) TLSConfig() (*tls.Config, error) {
	return f()
}

// A NonCA verifier picks up a certificate from the file system, rather than
// issuing one to itself, and serves it.
type NonCA struct{}

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

// TLSConfig initializes a server configuration that requires client TLS
// authentication based on a single certificate in the filesystem.
func (NonCA) TLSConfig() (*tls.Config, error) {
	serverTLSCert, err := mtls.LeafCertificateFromFile()
	if err != nil {
		return nil, errors.Wrap(err, "tls conversion")
	}

	conf, err := config(serverTLSCert)
	if err != nil {
		return nil, err
	}
	// TODO(cg): Sensors should also issue creds to, and verify, their clients.
	// For the time being, we only verify that the client cert is from the central CA.
	conf.ClientAuth = tls.VerifyClientCertIfGiven
	return conf, nil
}

// DefaultTLSServerConfig returns the default TLS config for servers in StackRox
func DefaultTLSServerConfig(certPool *x509.CertPool, certs []tls.Certificate) *tls.Config {
	// Government clients require TLS >=1.2 and require that AES-256 be preferred over AES-128
	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ClientCAs:    certPool,
		Certificates: certs,
	}
	cfg.NextProtos = []string{"h2"}
	return cfg
}

func config(serverBundle tls.Certificate) (*tls.Config, error) {
	certPool, err := TrustedCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "CA cert")
	}

	// This is based on TLSClientAuthServerConfig from cfssl/transport.
	// However, we don't use enough of their ecosystem to fully use it yet.
	cfg := DefaultTLSServerConfig(certPool, []tls.Certificate{serverBundle})
	return cfg, nil
}
