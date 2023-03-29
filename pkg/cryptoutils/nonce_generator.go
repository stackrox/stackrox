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
	Nonce() (string, error)
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

func (g nonceGenerator) Nonce() (string, error) {
	buf := make([]byte, g.nonceByteLen)
	if _, err := io.ReadFull(g.randSrc, buf); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}
