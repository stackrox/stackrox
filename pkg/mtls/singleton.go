package mtls

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance *certificateParserWrapper
	log      = logging.LoggerForModule()
)

// certificateParserWrapper is a singleton that wraps the CertificateParser interface.
// This allows us to override the implementation of the CertificateParser interface for testing purposes.
// This override mechanism is protected in ReleaseBuilds, and it will only work for DevelopmentBuilds.
type certificateParserWrapper struct {
	parser CertificateParser
}

// Override the internal parser. This should only be used for testing.
func (c *certificateParserWrapper) Override(parser CertificateParser) {
	if buildinfo.ReleaseBuild {
		// This is not allowed in release builds
		log.Errorf("The CertificateParser should not be override in production")
		return
	}
	c.parser = parser
}

// CertificateParser defines an interface with the functions to parse certificates.
type CertificateParser interface {
	LeafCertificateFromFile() (tls.Certificate, error)
	CACert() (*x509.Certificate, []byte, error)
}

// certificateParserImpl is the implementation of the CertificateParser interface.
type certificateParserImpl struct {
}

// GetCertificateParser returns the certificateParserWrapper singleton.
func GetCertificateParser() *certificateParserWrapper {
	once.Do(func() {
		instance = &certificateParserWrapper{
			parser: &certificateParserImpl{},
		}
	})
	return instance
}

// LeafCertificateFromFile reads a tls.Certificate (including private key and cert).
// This is the implementation that will be called by the LeafCertificateFromFile function.
func (c *certificateParserImpl) LeafCertificateFromFile() (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFilePathSetting.Setting(), keyFilePathSetting.Setting())
}

// CACert reads the cert from the local file system and returns the cert and the DER encoding.
// This is the implementation that will be called by the CACerts function.
func (c *certificateParserImpl) CACert() (*x509.Certificate, []byte, error) {
	caCert, _, caCertDER, caCertErr := readCA()
	return caCert, caCertDER, caCertErr
}
