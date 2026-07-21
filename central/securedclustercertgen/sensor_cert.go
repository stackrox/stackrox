package securedclustercertgen

import (
	"crypto/x509"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

// SensorCertificateValidity returns NotBefore and NotAfter for the Sensor service certificate in the set.
func SensorCertificateValidity(certificates *storage.TypedServiceCertificateSet) (notBefore, notAfter time.Time, err error) {
	if certificates == nil {
		return time.Time{}, time.Time{}, errors.New("certificates set is nil")
	}
	for _, typedCert := range certificates.GetServiceCerts() {
		if typedCert.GetServiceType() != storage.ServiceType_SENSOR_SERVICE {
			continue
		}
		certPEM := typedCert.GetCert().GetCertPem()
		if len(certPEM) == 0 {
			return time.Time{}, time.Time{}, errors.New("sensor certificate PEM is empty")
		}
		cert, parseErr := helpers.ParseCertificatePEM(certPEM)
		if parseErr != nil {
			return time.Time{}, time.Time{}, errors.Wrap(parseErr, "parsing sensor certificate")
		}
		return cert.NotBefore, cert.NotAfter, nil
	}
	return time.Time{}, time.Time{}, errors.New("sensor certificate not found in certificate set")
}

// ParseCertificateNotAfter parses NotAfter from a PEM-encoded certificate.
func ParseCertificateNotAfter(certPEM []byte) (time.Time, error) {
	cert, err := helpers.ParseCertificatePEM(certPEM)
	if err != nil {
		return time.Time{}, err
	}
	return cert.NotAfter, nil
}

// ParseCertificateValidity parses NotBefore and NotAfter from a PEM-encoded certificate.
func ParseCertificateValidity(certPEM []byte) (notBefore, notAfter time.Time, err error) {
	var cert *x509.Certificate
	cert, err = helpers.ParseCertificatePEM(certPEM)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return cert.NotBefore, cert.NotAfter, nil
}
