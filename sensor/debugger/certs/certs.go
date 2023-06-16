package certs

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/stackrox/rox/generated/storage"
)

type fakeCertificateParser struct {
	clusterID string
}

// LeafCertificateFromFile returns an empty tls.Certificate.
func (c *fakeCertificateParser) LeafCertificateFromFile() (tls.Certificate, error) {
	return tls.Certificate{}, nil
}

// CACert returns an empty x509.Certificate.
func (c *fakeCertificateParser) CACert() (*x509.Certificate, []byte, error) {
	return &x509.Certificate{}, []byte{}, nil
}

// ParseClusterIDFromServiceCert returns a dummy cluster id.
func (c *fakeCertificateParser) ParseClusterIDFromServiceCert(_ storage.ServiceType) (string, error) {
	return c.clusterID, nil
}

// WithClusterID sets the clusterID.
func (c *fakeCertificateParser) WithClusterID(clusterID string) *fakeCertificateParser {
	c.clusterID = clusterID
	return c
}

// NewSensorFakeCertsParser creates a new fakeCertificateParser.
func NewSensorFakeCertsParser() *fakeCertificateParser {
	return &fakeCertificateParser{
		clusterID: "00000000-0000-4000-A000-000000000000",
	}
}
