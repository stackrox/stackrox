package clusterid

import (
	"crypto/x509"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
)

var (
	sensorCNPrefix = fmt.Sprintf("%s: ", storage.ServiceType_SENSOR_SERVICE)
)

// ParseClusterIDFromServiceCert parses the service cert to extract cluster id
func ParseClusterIDFromServiceCert() (string, error) {
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

	if !strings.HasPrefix(cert.Subject.CommonName, sensorCNPrefix) {
		return "", errors.New("Malformed CN in certificate, unable to parse")
	}
	return strings.TrimPrefix(cert.Subject.CommonName, sensorCNPrefix), nil
}
