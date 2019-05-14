package verifier

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"
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

func config(serverBundle tls.Certificate) (*tls.Config, error) {
	certPool, err := TrustedCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "CA cert")
	}

	// This is based on TLSClientAuthServerConfig from cfssl/transport.
	// However, we don't use enough of their ecosystem to fully use it yet.
	return &tls.Config{
		Certificates: []tls.Certificate{serverBundle},
		ClientCAs:    certPool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
