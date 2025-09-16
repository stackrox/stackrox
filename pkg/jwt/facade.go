package jwt

import (
	"crypto/rsa"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// CreateRS256SignerAndValidator creates a token signer and validator pair with the given properties from the
// specified RSA private key.
func CreateRS256SignerAndValidator(issuerID string, audience jwt.Audience, key *rsa.PrivateKey, keyID string) (jose.Signer, Validator, error) {
	keyStore := NewSingleKeyStore(key.Public(), keyID)
	validator := NewRS256Validator(keyStore, issuerID, audience)
	signingKey := jose.SigningKey{
		Algorithm: jose.RS256,
		Key:       key,
	}
	signer, err := jose.NewSigner(signingKey, new(jose.SignerOptions).WithType("JWT").WithHeader("kid", keyID))
	if err != nil {
		return nil, nil, err
	}
	return signer, validator, nil
}

// CreateES256Validator creates a token validator pair with the given properties and jwks public key url
func CreateES256Validator(issuerID string, audience jwt.Audience, publicKeyURL string) (Validator, error) {
	keyStore := NewJWKSGetter(publicKeyURL)
	return NewES256Validator(keyStore, issuerID, audience), nil
}
