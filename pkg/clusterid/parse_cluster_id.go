package clusterid

import (
	"crypto/x509"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/mtls"
)

// ParseClusterIDFromServiceCert parses the service cert to extract cluster id.
// expectedServiceType specifies an optional service type expected for this cert. Use UNKNOWN_SERVICE
// for no expectation.
func ParseClusterIDFromServiceCert(expectedServiceType storage.ServiceType) (string, error) {
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
