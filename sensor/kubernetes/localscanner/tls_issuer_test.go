package localscanner

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

func TestHandler(t *testing.T) {
	suite.Run(t, new(tlsIssuerSuite))
}

type tlsIssuerSuite struct {
	suite.Suite
}

func (s *tlsIssuerSuite) TestGetScannerSecretDurationFromCertificate() {
	now := time.Now()
	afterOffset := 2 * 24 * time.Hour
	scannerCert := &x509.Certificate{
		NotBefore: now,
		NotAfter:  now.Add(afterOffset),
	}
	certDuration := getScannerSecretDurationFromCertificate(scannerCert)
	s.LessOrEqual(certDuration, afterOffset/2)
}
