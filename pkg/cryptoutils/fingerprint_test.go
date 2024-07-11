package cryptoutils

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"testing"
)

func BenchmarkCertFingerprintChoices(b *testing.B) {
	fakeCert := make([]byte, 400) // size of a normal cert
	n, err := rand.Read(fakeCert)
	if n < len(fakeCert) || err != nil {
		b.Fatalf("Expected %d bytes of randomness but got %d with error %v", len(fakeCert), n, err)
	}
	b.Run("SHA-1", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			sum := sha1.Sum(fakeCert)
			_ = formatID(sum[:])
		}
	})
	b.Run("SHA-256", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			sum := sha256.Sum256(fakeCert)
			_ = formatID(sum[:])
		}
	})
	b.Run("SHA-512", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			sum := sha512.Sum512(fakeCert)
			_ = formatID(sum[:])
		}
	})
	b.Run("SHA-512_256", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			sum := sha512.Sum512_256(fakeCert)
			_ = formatID(sum[:])
		}
	})
}
