package clusterid

import (
	"crypto/x509"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance *parserWrapper
	log      = logging.LoggerForModule()
)

// Parser defines an interface with the function to parse the cluster ID from the service cert.
type Parser interface {
	ParseClusterIDFromServiceCert(expectedServiceType storage.ServiceType) (string, error)
}

// parserWrapper is a singleton that wraps the Parser interface.
// This allows us to override the implementation of the Parser interface for testing purposes.
// This override mechanism is protected in ReleaseBuilds, and it will only work for DevelopmentBuilds.
type parserWrapper struct {
	parser Parser
}

// parserImpl is the implementation of the Parser interface.
type parserImpl struct {
}

// GetParser returns the parserWrapper singleton.
func GetParser() *parserWrapper {
	once.Do(func() {
		instance = &parserWrapper{
			parser: &parserImpl{},
		}
	})
	return instance
}

// Override the internal parser. This should only be used for testing.
func (p *parserWrapper) Override(parser Parser) {
	if buildinfo.ReleaseBuild {
		// This is not allowed in release builds
		log.Errorf("ParseClusterIDFromServiceCert should not be override in production")
		return
	}
	p.parser = parser
}

// ParseClusterIDFromServiceCert parses the service cert to extract cluster id.
// expectedServiceType specifies an optional service type expected for this cert. Use UNKNOWN_SERVICE
// for no expectation.
// This is the implementation that will be called by the ParseClusterIDFromServiceCert function.
func (p *parserImpl) ParseClusterIDFromServiceCert(expectedServiceType storage.ServiceType) (string, error) {
	leaf, err := mtls.LeafCertificateFromFile()
	if err != nil {
		return "", errors.Wrap(err, "Could not read sensor certificate")
	}

	if len(leaf.Certificate) == 0 {
		return "", errors.New("Malformed certificate, unable to parse")
	}

	cert, err := x509.ParseCertificate(leaf.Certificate[0])
	if err != nil {
		return "", errors.Wrap(err, "Unable to parse sensor certificate")
	}

	subj := mtls.SubjectFromCommonName(cert.Subject.CommonName)
	if expectedServiceType != storage.ServiceType_UNKNOWN_SERVICE && subj.ServiceType != expectedServiceType {
		return "", errors.Errorf("unexpected service type in cert: %v, expected %v", subj.ServiceType, expectedServiceType)
	}

	return subj.Identifier, nil
}

// ParseClusterIDFromServiceCert parses the service cert to extract cluster id.
// expectedServiceType specifies an optional service type expected for this cert. Use UNKNOWN_SERVICE
// for no expectation.
// We keep this function to avoid changing the client code.
func ParseClusterIDFromServiceCert(expectedServiceType storage.ServiceType) (string, error) {
	return GetParser().parser.ParseClusterIDFromServiceCert(expectedServiceType)
}
