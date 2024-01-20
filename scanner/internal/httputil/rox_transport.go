package httputil

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/mtls"
)

// defaultDialer is copied from http.DefaultTransport as of go1.20.10.
var defaultDialer = net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

// RoxTransportOptions represents transport options for reaching out to Rox-related services.
type RoxTransportOptions struct {
	disableCompression bool
}

// RoxTransport returns a [http.RoundTripper] capable of reaching out to a Rox-related service via mTLS.
func RoxTransport(subject mtls.Subject, o RoxTransportOptions) (http.RoundTripper, error) {
	//tlsConfig, err := clientconn.TLSConfig(subject, clientconn.TLSConfigOptions{
	//	UseClientCert: clientconn.MustUseClientCert,
	//})
	tlsConfig, err := TLSClientConfigForCentral()

	if err != nil {
		return nil, err
	}
	return &http.Transport{
		Proxy:              proxy.FromConfig(),
		TLSClientConfig:    tlsConfig,
		DisableCompression: o.disableCompression,

		// The rest are (more-or-less) copied from http.DefaultTransport as of go1.20.10.
		DialContext:           defaultDialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}, nil
}

var (
	certsPrefix = "/run/secrets/stackrox.io/certs/"

	// caCertFilePath is where the certificate is stored.
	caCertFilePath = certsPrefix + "ca.pem"

	// CertFilePath is where the certificate is stored.
	certFilePath = certsPrefix + "cert.pem"
	// KeyFilePath is where the key is stored.
	keyFilePath = certsPrefix + "key.pem"
	centralHostname = "central.stackrox"
)

var (
	readCAOnce sync.Once
	caCert     *x509.Certificate
	caCertDER  []byte
	caCertErr  error
)

// leafCertificateFromFile reads a tls.Certificate (including private key and cert)
// from the canonical locations on non-central services.
func leafCertificateFromFile() (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFilePath, keyFilePath)
}

// loadCACertDER reads the PEM-decoded bytes of the cert from the local file system.
func loadCACertDER() ([]byte, error) {
	b, err := os.ReadFile(caCertFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "file access")
	}
	decoded, _ := pem.Decode(b)
	if decoded == nil {
		return nil, errors.New("invalid PEM")
	}
	return decoded.Bytes, nil
}

// readCACert reads the cert from the local file system and returns the cert and the DER encoding.
func readCACert() (*x509.Certificate, []byte, error) {
	readCAOnce.Do(func() {
		der, err := loadCACertDER()
		if err != nil {
			caCertErr = errors.Wrap(err, "CA cert could not be decoded")
			return
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			caCertErr = errors.Wrap(err, "CA cert could not be parsed")
			return
		}
		caCert = cert
		caCertDER = der
	})
	return caCert, caCertDER, caCertErr
}

// trustedCertPool creates a CertPool that contains the CA certificate.
func trustedCertPool() (*x509.CertPool, error) {
	caCert, _, err := readCACert()
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	certPool.AddCert(caCert)
	return certPool, nil
}

// TLSClientConfigForCentral returns a TLS client config that can be used to talk to Central.
func TLSClientConfigForCentral() (*tls.Config, error) {
	certPool, err := trustedCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "loading trusted cert pool")
	}
	leafCert, err := leafCertificateFromFile()
	if err != nil {
		return nil, errors.Wrap(err, "loading leaf cert")
	}
	conf := &tls.Config{
		ServerName: centralHostname,
		Certificates: []tls.Certificate{
			leafCert,
		},
		RootCAs: certPool,
	}
	return conf, nil
}
