package certwatch

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestCert(t *testing.T, serialNumber int64) tls.Certificate {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(serialNumber),
		Subject: pkix.Name{
			CommonName: "test.example.com",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privateKey,
		Leaf:        cert,
	}
}

func TestUpdateTLSConfig(t *testing.T) {
	certs := []tls.Certificate{generateTestCert(t, 1)}
	holder := NewTLSConfigHolder(&tls.Config{MinVersion: tls.VersionTLS12}, tls.NoClientCert)
	holder.AddServerCertSource(&certs)

	holder.UpdateTLSConfig()
	liveCfg1 := holder.liveTLSConfig.Load()
	require.NotNil(t, liveCfg1)
	assert.Equal(t, int64(1), liveCfg1.Certificates[0].Leaf.SerialNumber.Int64())

	certs[0] = generateTestCert(t, 2)
	holder.UpdateTLSConfig()

	liveCfg2 := holder.liveTLSConfig.Load()
	require.NotNil(t, liveCfg2)
	assert.Equal(t, int64(2), liveCfg2.Certificates[0].Leaf.SerialNumber.Int64())
	assert.NotSame(t, liveCfg1, liveCfg2)
}

func TestUpdateTLSConfigRotatesSessionTicketKeys(t *testing.T) {
	originalRotator := sessionTicketKeyRotator
	defer func() { sessionTicketKeyRotator = originalRotator }()

	var rotationCalled bool
	var rotatedConfig *tls.Config
	sessionTicketKeyRotator = func(cfg *tls.Config) error {
		rotationCalled = true
		rotatedConfig = cfg
		return nil
	}

	certs := []tls.Certificate{generateTestCert(t, 1)}
	holder := NewTLSConfigHolder(&tls.Config{MinVersion: tls.VersionTLS12}, tls.NoClientCert)
	holder.AddServerCertSource(&certs)

	holder.UpdateTLSConfig()

	assert.True(t, rotationCalled)
	assert.NotNil(t, rotatedConfig)
}
