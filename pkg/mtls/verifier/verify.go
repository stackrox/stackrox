package verifier

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	"bitbucket.org/stack-rox/apollo/pkg/mtls"
	"bitbucket.org/stack-rox/apollo/pkg/tls/keys"
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

// A NoMTLS verifier generates a random certificate without using a trusted CA at all.
// Deprecated: This should only be used until MTLS is supported on all platforms.
type NoMTLS struct{}

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

// TLSConfig initializes a server configuration that uses a randomly generated
// key and does not verify client certs.
func (NoMTLS) TLSConfig() (*tls.Config, error) {
	pool, cert, err := getPair()
	if err != nil {
		return nil, fmt.Errorf("tls conversion: %s", err)
	}

	return &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{*cert},
	}, nil
}

func getPair() (*x509.CertPool, *tls.Certificate, error) {
	cert, key, err := keys.GenerateStackRoxKeyPair()
	if err != nil {
		return nil, nil, err
	}
	pair, err := tls.X509KeyPair(cert.Key().PEM(), key.Key().PEM())
	if err != nil {
		return nil, nil, err
	}
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(cert.Key().PEM())
	if !ok {
		return nil, nil, errors.New("Cert is invalid")
	}
	return pool, &pair, nil
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
