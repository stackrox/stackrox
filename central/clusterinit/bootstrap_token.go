package clusterinit

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"github.com/pkg/errors"
)

const (
	bootstrapTokenIDByteLength int = 4
	bootstrapTokenByteLength   int = 32 // Suitable for use with AES-256
)

// BootstrapToken models a single bootstrap token without any metadata.
type BootstrapToken []byte

// ID returns the fingerprint of the token.
func (t BootstrapToken) ID() string {
	hash := sha256.Sum256([]byte(t))
	return fmt.Sprintf("%x", hash[:bootstrapTokenIDByteLength])
}

// GenerateBootstrapToken generates a new bootstrap token.
func GenerateBootstrapToken() (BootstrapToken, error) {
	bytes := make([]byte, bootstrapTokenByteLength)

	_, err := rand.Read(bytes)
	if err != nil {
		return nil, errors.Wrap(err, "generating token")
	}

	token := BootstrapToken(bytes)

	return token, nil
}
