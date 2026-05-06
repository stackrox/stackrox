package verifier

import (
	"crypto/tls"
	"crypto/x509"
	"path/filepath"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/certwatch"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/tlsprofile"
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

var (
	leafCert     atomic.Pointer[tls.Certificate]
	leafCertOnce sync.Once
)

func loadAndWatchLeafCert() {
	leafCertOnce.Do(func() {
		certwatch.WatchCertDir("service", filepath.Dir(mtls.CertFilePath()),
			loadLeafCertFromDirectory, func(cert *tls.Certificate) {
				if cert != nil {
					leafCert.Store(cert)
				}
			}, certwatch.WithVerify(false))
	})
}

func loadLeafCertFromDirectory(dir string) (*tls.Certificate, error) {
	certFile := filepath.Join(dir, mtls.ServiceCertFileName)
	keyFile := filepath.Join(dir, mtls.ServiceKeyFileName)

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Wrapf(err, "loading service certificate from %q", dir)
	}

	return &cert, nil
}

// TrustedCertPool creates a CertPool that contains the CA certificate.
func TrustedCertPool() (*x509.CertPool, error) {
	caCert, _, err := mtls.CACert()
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	certPool.AddCert(caCert)
	addSecondaryCACertIfExists(certPool)
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
	addSecondaryCACertIfExists(certPool)
	return certPool, nil
}

func addSecondaryCACertIfExists(certPool *x509.CertPool) {
	secondaryCACert, _, err := mtls.SecondaryCACert()
	if err == nil {
		certPool.AddCert(secondaryCACert)
	}
}

// TLSConfig initializes a server configuration that requires client TLS
// authentication based on a single certificate in the filesystem.
// The returned config uses GetConfigForClient to serve the latest cert
// from a file watcher, enabling hot reload when cert files change on disk.
func (NonCA) TLSConfig() (*tls.Config, error) {
	loadAndWatchLeafCert()

	rootConf, err := serverTLSConfig()
	if err != nil {
		return nil, err
	}

	rootConf.ClientAuth = tls.VerifyClientCertIfGiven
	rootConf.GetCertificate = func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
		cert := leafCert.Load()
		if cert == nil {
			return nil, errors.New("no leaf certificate available")
		}
		return cert, nil
	}

	return rootConf, nil
}

// DefaultTLSServerConfig returns the default TLS config for servers in StackRox.
//
// The minimum TLS version and cipher suites can be overridden via the
// ROX_TLS_MIN_VERSION and ROX_TLS_CIPHER_SUITES environment variables.
// When these are unset the compiled-in defaults are used (TLS 1.2 with
// AES-256-GCM preferred over AES-128-GCM).
func DefaultTLSServerConfig(certPool *x509.CertPool, certs []tls.Certificate) *tls.Config {
	cfg := &tls.Config{
		MinVersion:               tlsprofile.MinVersion(),
		PreferServerCipherSuites: true,
		CipherSuites:             tlsprofile.CipherSuites(),
		ClientAuth:               tls.VerifyClientCertIfGiven,
		ClientCAs:                certPool,
		Certificates:             certs,
	}
	cfg.NextProtos = []string{"h2"}
	return cfg
}

func serverTLSConfig() (*tls.Config, error) {
	certPool, err := TrustedCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "CA cert")
	}
	return DefaultTLSServerConfig(certPool, nil), nil
}
