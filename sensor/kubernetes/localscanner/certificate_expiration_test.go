package localscanner

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetSecretRenewalTimeFromCertificate(t *testing.T) {
	now := time.Now()
	afterOffset := 2 * 24 * time.Hour
	scannerCert := &x509.Certificate{
		NotBefore: now,
		NotAfter:  now.Add(afterOffset),
	}
	certRenewalTime := getSecretRenewalTimeFromCertificate(scannerCert)
	certDuration := time.Until(certRenewalTime)
	assert.LessOrEqual(t, certDuration, afterOffset/2)
}