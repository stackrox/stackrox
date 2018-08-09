package verifier

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/stackrox/rox/pkg/mtls"
)

// A TLSConfigurer instantiates the appropriate TLS config for your environment.
type TLSConfigurer interface {
	TLSConfig() (*tls.Config, error)
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
	serverCert, serverKey, _, err := mtls.IssueNewCert(mtls.CentralCN, nil)
	if err != nil {
		return nil, fmt.Errorf("server keypair: %s", err)
	}
	caPEM, err := mtls.CACertPEM()
	if err != nil {
		return nil, fmt.Errorf("CA cert retrieval: %s", err)
	}
	serverCertBundle := append(serverCert, caPEM...)

	serverTLSCert, err := tls.X509KeyPair(serverCertBundle, serverKey)
	if err != nil {
		return nil, fmt.Errorf("tls conversion: %s", err)
	}

	return config(serverTLSCert)
}

// TLSConfig initializes a server configuration that requires client TLS
// authentication based on the Certificate Authority we are using.
// TODO(cg): NonCA currently does not verify the client cert.
func (NonCA) TLSConfig() (*tls.Config, error) {
	serverTLSCert, err := mtls.LeafCertificateFromFile()
	if err != nil {
		return nil, fmt.Errorf("tls conversion: %s", err)
	}

	conf, err := config(serverTLSCert)
	if err != nil {
		return nil, err
	}
	// TODO(cg): Sensors should also issue creds to, and verify, their clients.
	conf.ClientAuth = tls.NoClientCert
	return conf, nil
}

func config(serverBundle tls.Certificate) (*tls.Config, error) {
	certPool, err := TrustedCertPool()
	if err != nil {
		return nil, fmt.Errorf("CA cert: %s", err)
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
