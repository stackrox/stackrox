package x509utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestCert(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}

func TestParseCertificatePEM(t *testing.T) {
	certPEM, _ := generateTestCert(t)

	cases := map[string]struct {
		input   []byte
		wantErr bool
	}{
		"valid cert": {
			input: certPEM,
		},
		"nil input": {
			input:   nil,
			wantErr: true,
		},
		"empty input": {
			input:   []byte{},
			wantErr: true,
		},
		"invalid PEM": {
			input:   []byte("not a pem block"),
			wantErr: true,
		},
		"invalid DER inside PEM": {
			input:   pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("garbage")}),
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cert, err := ParseCertificatePEM(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cert)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "test", cert.Subject.CommonName)
			}
		})
	}
}

func TestParseCertificatesPEM(t *testing.T) {
	cert1PEM, _ := generateTestCert(t)
	cert2PEM, _ := generateTestCert(t)

	cases := map[string]struct {
		input     []byte
		wantCount int
		wantErr   bool
	}{
		"single cert": {
			input:     cert1PEM,
			wantCount: 1,
		},
		"two certs": {
			input:     append(cert1PEM, cert2PEM...),
			wantCount: 2,
		},
		"skips non-CERTIFICATE blocks": {
			input: append(
				pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("key")}),
				cert1PEM...,
			),
			wantCount: 1,
		},
		"no certs found": {
			input:   pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("key")}),
			wantErr: true,
		},
		"empty input": {
			input:   []byte{},
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			certs, err := ParseCertificatesPEM(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, certs, tc.wantCount)
			}
		})
	}
}

func TestParsePrivateKeyPEM(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	ecPKCS8, err := x509.MarshalPKCS8PrivateKey(ecKey)
	require.NoError(t, err)
	ecDER, err := x509.MarshalECPrivateKey(ecKey)
	require.NoError(t, err)

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	rsaPKCS1 := x509.MarshalPKCS1PrivateKey(rsaKey)
	rsaPKCS8, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	require.NoError(t, err)

	cases := map[string]struct {
		input   []byte
		wantErr bool
	}{
		"ECDSA PKCS8": {
			input: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: ecPKCS8}),
		},
		"ECDSA SEC1": {
			input: pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: ecDER}),
		},
		"RSA PKCS8": {
			input: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: rsaPKCS8}),
		},
		"RSA PKCS1": {
			input: pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: rsaPKCS1}),
		},
		"invalid PEM": {
			input:   []byte("not a pem"),
			wantErr: true,
		},
		"invalid key data": {
			input:   pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("garbage")}),
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			signer, err := ParsePrivateKeyPEM(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, signer)
			}
		})
	}
}
