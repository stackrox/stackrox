package jwt

import (
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// SignerFactory produces jose.Signer on demand.
type SignerFactory struct {
	keyStore PrivateKeyGetter
	keyID    string
}

// CreateSigner creates jose.Signer based on underlying private key.
func (f *SignerFactory) CreateSigner() (jose.Signer, error) {
	signingKey := jose.SigningKey{
		Algorithm: jose.RS256,
		Key:       f.keyStore.Key(f.keyID),
	}
	return jose.NewSigner(signingKey, new(jose.SignerOptions).WithType("JWT").WithHeader("kid", f.keyID))
}

// CreateRS256Validator creates a token validator with the given properties from the
// specified RSA public key.
func CreateRS256Validator(issuerID string, audience jwt.Audience, publicKeyStore PublicKeyGetter) Validator {
	return NewRS256Validator(publicKeyStore, issuerID, audience)
}

// CreateRS256SignerFactory creates a token signer factory with the given properties from the
// specified RSA private key.
func CreateRS256SignerFactory(privateKeyStore PrivateKeyGetter, keyID string) *SignerFactory {
	return &SignerFactory{
		keyStore: privateKeyStore,
		keyID:    keyID,
	}
}

// CreateES256Validator creates a token validator pair with the given properties and jwks public key url
func CreateES256Validator(issuerID string, audience jwt.Audience, publicKeyURL string) (Validator, error) {
	keyStore := NewJWKSGetter(publicKeyURL)
	return NewES256Validator(keyStore, issuerID, audience), nil
}
