package cryptoutils

import (
	"crypto/sha512"
	"crypto/x509"
)

// CertFingerprint computes the fingerprint of a certificate using SHA-512_256.
func CertFingerprint(cert *x509.Certificate) string {
	// sha512 is actually faster than 256 on 64 bit architectures
	fingerprint := sha512.Sum512_256(cert.Raw)
	return formatID(fingerprint[:])
}
