package localscanner

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetSecretRenewalTimeFromCertificate(t *testing.T) {
	beforeTime := time.Unix(0, 0)
	afterOffset := 2 * 24 * time.Hour
	scannerCert := &x509.Certificate{
		NotBefore: beforeTime,
		NotAfter:  beforeTime.Add(afterOffset),
	}
	certRenewalTime := getSecretRenewalTimeFromCertificate(scannerCert)
	certDuration := time.Until(certRenewalTime)
	assert.LessOrEqual(t, certDuration, afterOffset/2)
}
