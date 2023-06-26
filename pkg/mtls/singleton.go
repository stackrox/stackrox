package mtls

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance CertificateParser
	log      = logging.LoggerForModule()
	mu       sync.Mutex
)

// CertificateParser defines an interface with the functions to parse certificates.
type CertificateParser interface {
	LeafCertificateFromFile() (tls.Certificate, error)
	CACert() (*x509.Certificate, []byte, error)
}

// certificateParserImpl is the implementation of the CertificateParser interface.
type certificateParserImpl struct {
}

// GetCertificateParser returns the CertificateParser singleton.
func GetCertificateParser() CertificateParser {
	mu.Lock()
	defer mu.Unlock()
	once.Do(func() {
		instance = &certificateParserImpl{}
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
