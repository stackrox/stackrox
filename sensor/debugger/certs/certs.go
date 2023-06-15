package certs

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/stackrox/rox/generated/storage"
)

type certificateParser struct {
}

// LeafCertificateFromFile returns an empty tls.Certificate.
func (c *certificateParser) LeafCertificateFromFile() (tls.Certificate, error) {
	return tls.Certificate{}, nil
}

// CACert returns an empty x509.Certificate.
func (c *certificateParser) CACert() (*x509.Certificate, []byte, error) {
	return &x509.Certificate{}, []byte{}, nil
}

// ParseClusterIDFromServiceCert returns a dummy cluster id.
func (c *certificateParser) ParseClusterIDFromServiceCert(_ storage.ServiceType) (string, error) {
	return "00000000-0000-4000-A000-000000000000", nil
}

// NewSensorCertsParser creates a new SensorCertsParser
func NewSensorCertsParser() *certificateParser {
	return &certificateParser{}
}
