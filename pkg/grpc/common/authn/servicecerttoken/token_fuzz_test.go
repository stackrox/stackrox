package servicecerttoken

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

// FuzzParseToken tests that ParseToken does not panic on arbitrary string input.
//
// The function parses a two-part token (base64.base64) containing:
// 1. Serialized ServiceCertAuth proto with cert DER and timestamp
// 2. Cryptographic signature
//
// The fuzzer ensures robustness against:
// - Malformed token formats (wrong number of parts, missing dots, etc.)
// - Invalid base64 encoding
// - Corrupted protobuf data
// - Invalid certificate DER data
// - Out-of-range timestamps
// - Invalid signatures
// - Empty or null inputs
func FuzzParseToken(f *testing.F) {
	// Generate a valid certificate for seed corpus
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		f.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			CommonName:   "test.stackrox.io",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		f.Fatalf("failed to create certificate: %v", err)
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}

	// Seed corpus: valid token
	validToken, err := CreateToken(cert, time.Now())
	if err != nil {
		f.Fatalf("failed to create valid token: %v", err)
	}
	f.Add(validToken)

	// Seed corpus: valid token with time in the past
	pastToken, err := CreateToken(cert, time.Now().Add(-5*time.Second))
	if err == nil {
		f.Add(pastToken)
	}

	// Seed corpus: valid token with time in the future
	futureToken, err := CreateToken(cert, time.Now().Add(5*time.Second))
	if err == nil {
		f.Add(futureToken)
	}

	// Seed corpus: manually crafted proto with minimal valid data
	ts, _ := protocompat.ConvertTimeToTimestampOrError(time.Now())
	auth := &central.ServiceCertAuth{
		CertDer:     certDER,
		CurrentTime: ts,
	}
	authBytes, _ := auth.MarshalVT()
	minimalToken := base64.RawStdEncoding.EncodeToString(authBytes) + "." + base64.RawStdEncoding.EncodeToString([]byte("invalid-sig"))
	f.Add(minimalToken)

	// Seed corpus: edge cases for token format
	f.Add("")                // empty
	f.Add(".")               // single dot
	f.Add("..")              // double dot
	f.Add("...")             // triple dot
	f.Add("a.b")             // valid format but invalid base64/data
	f.Add("a.b.c")           // too many parts
	f.Add("YWJj")            // no separator
	f.Add("YWJj.")           // missing second part
	f.Add(".YWJj")           // missing first part
	f.Add("!@#$%^&*.(){}[]") // invalid characters
	f.Add("YWJj.ZGVm")       // valid base64, invalid proto
	f.Add("AAAA.AAAA")       // valid base64, null bytes

	// Seed corpus: base64 edge cases
	f.Add("==.==")          // padding characters
	f.Add("YQ==.Yg==")      // padded base64
	f.Add("YWJjZA.ZGVmZ2g") // valid raw base64

	// Seed corpus: very long strings (potential buffer issues)
	longPart := base64.RawStdEncoding.EncodeToString(make([]byte, 10000))
	f.Add(longPart + "." + longPart)

	// Seed corpus: special characters and unicode
	f.Add("测试.测试")             // unicode
	f.Add("test\x00null.test") // null bytes
	f.Add("test\nline.break")  // newlines

	// Seed corpus: corrupted proto data
	corruptedAuth := authBytes
	if len(corruptedAuth) > 5 {
		corruptedAuth[3] = ^corruptedAuth[3] // flip some bits
	}
	corruptedToken := base64.RawStdEncoding.EncodeToString(corruptedAuth) + "." + base64.RawStdEncoding.EncodeToString([]byte("sig"))
	f.Add(corruptedToken)

	// Seed corpus: invalid timestamp proto
	authBadTime := &central.ServiceCertAuth{
		CertDer:     certDER,
		CurrentTime: nil, // nil timestamp
	}
	authBadTimeBytes, _ := authBadTime.MarshalVT()
	badTimeToken := base64.RawStdEncoding.EncodeToString(authBadTimeBytes) + "." + base64.RawStdEncoding.EncodeToString([]byte("sig"))
	f.Add(badTimeToken)

	// Seed corpus: invalid cert DER
	authBadCert := &central.ServiceCertAuth{
		CertDer:     []byte("not-a-certificate"),
		CurrentTime: ts,
	}
	authBadCertBytes, _ := authBadCert.MarshalVT()
	badCertToken := base64.RawStdEncoding.EncodeToString(authBadCertBytes) + "." + base64.RawStdEncoding.EncodeToString([]byte("sig"))
	f.Add(badCertToken)

	// Seed corpus: empty cert DER
	authEmptyCert := &central.ServiceCertAuth{
		CertDer:     []byte{},
		CurrentTime: ts,
	}
	authEmptyCertBytes, _ := authEmptyCert.MarshalVT()
	emptyCertToken := base64.RawStdEncoding.EncodeToString(authEmptyCertBytes) + "." + base64.RawStdEncoding.EncodeToString([]byte("sig"))
	f.Add(emptyCertToken)

	// Run fuzzer
	f.Fuzz(func(t *testing.T, token string) {
		// The fuzzer's goal is to ensure no panics occur, regardless of input.
		// We don't care if parsing succeeds or fails, only that it doesn't panic.
		// Test with various maxLeeway values to exercise different code paths
		assert.NotPanics(t, func() {
			_, _ = ParseToken(token, 0)
		})

		assert.NotPanics(t, func() {
			_, _ = ParseToken(token, 10*time.Second)
		})

		assert.NotPanics(t, func() {
			_, _ = ParseToken(token, 24*time.Hour)
		})

		// If we reach here without panic, the test passes for this input.
	})
}
