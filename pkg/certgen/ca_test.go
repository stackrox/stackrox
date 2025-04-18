package certgen

import (
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCA(t *testing.T) {
	ca, err := GenerateCA()
	assert.NoError(t, err)

	certPEM := ca.CertPEM()
	block, _ := pem.Decode(certPEM)
	assert.NotNil(t, block)

	cert, err := x509.ParseCertificate(block.Bytes)
	assert.NoError(t, err)

	const fiveYears = 5 * 365 * 24 * time.Hour
	validity := cert.NotAfter.Sub(cert.NotBefore)
	assert.InDelta(t, fiveYears, validity, float64(time.Hour), "cert validity should be ~5 years")
}
