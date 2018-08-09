package signer

import (
	"crypto/x509"
	"time"

	"github.com/stackrox/rox/pkg/uuid"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// A Signer returns signed JWTs. It's a thin wrapper that passes all of Central's constant parameters
// to go-jose's signer.
type Signer interface {
	// SignedJWT returns a signed JWT with the subject field set to the given value.
	// It also returns metadata associated with the given token.
	SignedJWT(subject string) (token, id string, issuedAt, expiration time.Time, err error)
	// Key returns the public (NOT the private) key.
	// The Signer is strict about the id. It returns nil, false
	// unless the id passed in matches the signer's key's id.
	// This implements the KeyGetter interface.
	Key(id string) (interface{}, bool)
}

// NewFromBytes returns a new signer from (DER-format) bytes.
func NewFromBytes(privateKeyBytes []byte) (Signer, error) {
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBytes)
	if err != nil {
		return nil, err
	}
	keyID := uuid.NewV4()
	sig, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: privateKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", keyID.String()),
	)
	if err != nil {
		return nil, err
	}

	return &signerImpl{keyID: keyID, key: privateKey, builder: jwt.Signed(sig)}, nil
}
