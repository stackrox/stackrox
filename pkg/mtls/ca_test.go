package mtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCA creates a self-signed CA for testing (replaces cfssl/initca).
func newTestCA(t *testing.T, cn string) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}

func Test_CA_IssueCertForSubject(t *testing.T) {
	tests := map[string]struct {
		opts        []IssueCertOption
		minNotAfter time.Duration
		maxNotAfter time.Duration
	}{
		"regular cert": {
			opts:        nil,
			minNotAfter: 364 * 24 * time.Hour,
			maxNotAfter: 366 * 24 * time.Hour,
		},
		"ephemeral cert hourly expiration": {
			opts:        []IssueCertOption{WithValidityExpiringInHours()},
			minNotAfter: 2 * time.Hour,
			maxNotAfter: 4 * time.Hour,
		},
		"ephemeral cert daily expiration": {
			opts:        []IssueCertOption{WithValidityExpiringInDays()},
			minNotAfter: (2*24 - 1) * time.Hour,
			maxNotAfter: (2*24 + 1) * time.Hour,
		},
	}

	cert, key := newTestCA(t, "Fake CA")
	ca, err := LoadCAForSigning(cert, key)
	require.NoError(t, err)

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ca.IssueCertForSubject(CentralSubject, tt.opts...)
			require.NoError(t, err)

			notAfter := got.X509Cert.NotAfter
			assert.True(t, notAfter.After(time.Now().Add(tt.minNotAfter)), "expected notAfter=%q to be after %q from now", notAfter, tt.minNotAfter)
			assert.True(t, notAfter.Before(time.Now().Add(tt.maxNotAfter)), "expected notAfter=%q to be before %q from now", notAfter, tt.maxNotAfter)
		})
	}
}

func Test_CA_LoadForValidation(t *testing.T) {
	certPEM, keyPEM := newTestCA(t, "Fake CA")

	ca, err := LoadCAForValidation(certPEM)
	require.NoError(t, err)
	require.NotNil(t, ca.Certificate(), "expected CA certificate to be present")
	assert.Equal(t, "Fake CA", ca.Certificate().Subject.CommonName)

	certPool := ca.CertPool()
	require.NotNil(t, certPool, "expected non-nil certificate pool")

	assert.Nil(t, ca.PrivateKey(), "expected PrivateKey to be nil for validation-only CA")
	assert.Nil(t, ca.KeyPEM(), "expected KeyPEM to be nil for validation-only CA")

	_, err = ca.IssueCertForSubject(CentralSubject)
	require.Error(t, err, "expected signing to fail for validation-only CA")

	// issue a leaf certificate and validate it
	signingCA, err := LoadCAForSigning(certPEM, keyPEM)
	require.NoError(t, err)

	issuedCert, err := signingCA.IssueCertForSubject(CentralSubject)
	require.NoError(t, err)

	subject, err := ca.ValidateAndExtractSubject(issuedCert.X509Cert)
	require.NoError(t, err, "expected certificate to be valid")
	assert.Equal(t, CentralSubject, subject, "extracted subject should match issued one")

	// issue a leaf certificate with an unrelated CA and try to validate it
	unrelatedCertPEM, unrelatedKeyPEM := newTestCA(t, "Unrelated CA")

	unrelatedSigningCA, err := LoadCAForSigning(unrelatedCertPEM, unrelatedKeyPEM)
	require.NoError(t, err)

	unrelatedIssuedCert, err := unrelatedSigningCA.IssueCertForSubject(CentralSubject)
	require.NoError(t, err)

	_, err = ca.ValidateAndExtractSubject(unrelatedIssuedCert.X509Cert)
	require.Error(t, err, "expected validation to fail for unrelated CA")
}
