package verifier

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/mtls"
)

// A TLSConfigurer instantiates the appropriate TLS config for your environment.
type TLSConfigurer interface {
	TLSConfig() (*tls.Config, error)
}

// FirstWorkingConfigurer is a TLS configurer that takes the TLS config from the first working
// TLS configurer contained in the wrapped slice.
type FirstWorkingConfigurer []TLSConfigurer

// TLSConfig returns the config from the first TLS configurer in the wrapped slice that returned a
// non-error result.
func (c FirstWorkingConfigurer) TLSConfig() (*tls.Config, error) {
	if len(c) == 0 {
		return nil, errors.New("no TLS configurer specified")
	}

	errs := errorhelpers.NewErrorList("determining TLS configuration")
	for _, configurer := range c {
		cfg, err := configurer.TLSConfig()
		if err == nil {
			return cfg, nil
		}
		errs.AddError(err)
	}
	return nil, errs.ToError()
}

// A CA issues itself a certificate and serves it.
type CA struct{}

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
// authentication based on the Certificate Authority we are using.
func (CA) TLSConfig() (*tls.Config, error) {
	issuedCert, err := mtls.IssueNewCert(mtls.CentralSubject, nil)
	if err != nil {
		return nil, errors.Wrap(err, "server keypair")
	}
	caPEM, err := mtls.CACertPEM()
	if err != nil {
		return nil, errors.Wrap(err, "CA cert retrieval")
	}
	serverCertBundle := append(issuedCert.CertPEM, caPEM...)

	serverTLSCert, err := tls.X509KeyPair(serverCertBundle, issuedCert.KeyPEM)
	if err != nil {
		return nil, errors.Wrap(err, "tls conversion")
	}

	return config(serverTLSCert)
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
