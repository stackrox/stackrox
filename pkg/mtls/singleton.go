package mtls

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance *certificateParserWrapper
)

type certificateParserWrapper struct {
	parser CertificateParser
}

// Override the internal parser. This should only be used for testing.
func (c *certificateParserWrapper) Override(parser CertificateParser) {
	if buildinfo.ReleaseBuild {
		// This is not allowed in release builds
		return
	}
	c.parser = parser
}

// CertificateParser defines the functions to parse certificate.
type CertificateParser interface {
	LeafCertificateFromFile() (tls.Certificate, error)
	CACert() (*x509.Certificate, []byte, error)
}

type certificateParserImpl struct {
}

var _ CertificateParser = (*certificateParserImpl)(nil)

// GetCertificateParser returns the cryptoFactory
func GetCertificateParser() *certificateParserWrapper {
	once.Do(func() {
		instance = &certificateParserWrapper{
			parser: &certificateParserImpl{},
		}
	})
	return instance
}

// LeafCertificateFromFile reads a tls.Certificate (including private key and cert).
func (c *certificateParserImpl) LeafCertificateFromFile() (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFilePathSetting.Setting(), keyFilePathSetting.Setting())
}

// CACert reads the cert from the local file system and returns the cert and the DER encoding.
func (c *certificateParserImpl) CACert() (*x509.Certificate, []byte, error) {
	caCert, _, caCertDER, caCertErr := readCA()
	return caCert, caCertDER, caCertErr
}
