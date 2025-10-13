package cryptoutils

import (
	"crypto/rand"
	"encoding/base64"
	"io"
)

// NonceGenerator is a generator for cryptographically secure nonces.
//
//go:generate mockgen-wrapper
type NonceGenerator interface {
	// Nonce generates new nonce and returns it as base64 URL encoded string
	Nonce() (string, error)

	// NonceBytes generates new nonce and returns it as a slice of bytes
	NonceBytes() ([]byte, error)
}

// NewNonceGenerator creates a new nonce generator issuing base64 URL-encoded nonces with the given underlying
// byte length.
func NewNonceGenerator(nonceByteLen int, randSrc io.Reader) NonceGenerator {
	if randSrc == nil {
		randSrc = rand.Reader
	}
	return nonceGenerator{
		randSrc:      randSrc,
		nonceByteLen: nonceByteLen,
	}
}

type nonceGenerator struct {
	randSrc      io.Reader
	nonceByteLen int
}

// Nonce generates new nonce and returns it as base64 URL encoded string
func (g nonceGenerator) Nonce() (string, error) {
	buf := make([]byte, g.nonceByteLen)
	if _, err := io.ReadFull(g.randSrc, buf); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

// NonceBytes generates new nonce and returns it as a slice of bytes
func (g nonceGenerator) NonceBytes() ([]byte, error) {
	buf := make([]byte, g.nonceByteLen)
	if _, err := io.ReadFull(g.randSrc, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
