package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
)

const (
	tlsCertFileName = `tls.crt`
	tlsKeyFileName  = `tls.key`

	defaultCertPath = `/run/secrets/stackrox.io/default-tls-cert`
)

// NewCentralTLSConfigurer returns a new tls configurer to be used for Central.
func NewCentralTLSConfigurer() verifier.TLSConfigurer {
	return verifier.TLSConfigurerFunc(createTLSConfig)
}

func loadDefaultCertificate() (*tls.Certificate, error) {
	certFile := filepath.Join(defaultCertPath, tlsCertFileName)
	keyFile := filepath.Join(defaultCertPath, tlsKeyFileName)

	if filesExist, err := fileutils.AllExist(certFile, keyFile); err != nil || !filesExist {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, errors.Wrap(err, "parsing leaf certificate")
	}

	return &cert, nil
}

func loadInternalCertificateFromFiles() (*tls.Certificate, error) {
	if filesExist, err := fileutils.AllExist(mtls.CertFilePath, mtls.KeyFilePath); err != nil || !filesExist {
		return nil, err
	}

	cert, err := mtls.LeafCertificateFromFile()
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

func issueInternalCertificate() (*tls.Certificate, error) {
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
	return &serverTLSCert, nil
}

func getInternalCertificate() (*tls.Certificate, error) {
	// First try to load the internal certificate from files. If the files don't exist, issue
	// ourselves a cert.
	if certFromFiles, err := loadInternalCertificateFromFiles(); err != nil {
		return nil, err
	} else if certFromFiles != nil {
		return certFromFiles, nil
	}

	return issueInternalCertificate()
}

func createTLSConfig() (*tls.Config, error) {
	certPool, err := verifier.TrustedCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "loading trusted cert pool")
	}

	var certs []tls.Certificate

	defaultCert, err := loadDefaultCertificate()
	if err != nil {
		return nil, errors.Wrap(err, "loading default certificate")
	}
	if defaultCert != nil {
		certs = append(certs, *defaultCert)
	}

	internalCert, err := getInternalCertificate()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving internal certificate")
	} else if internalCert == nil {
		return nil, errors.New("no internal cert available")
	}
	certs = append(certs, *internalCert)

	cfg := verifier.DefaultTLSServerConfig(certPool, certs)
	cfg.BuildNameToCertificate()

	return cfg, nil
}
