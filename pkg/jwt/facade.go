package jwt

import (
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type SignerGetter struct {
	keyStore KeyGetter
	keyID    string
}

func (f *SignerGetter) GetSigner() (jose.Signer, error) {
	signingKey := jose.SigningKey{
		Algorithm: jose.RS256,
		Key:       f.keyStore.Key(f.keyID),
	}
	return jose.NewSigner(signingKey, new(jose.SignerOptions).WithType("JWT").WithHeader("kid", f.keyID))
}

// CreateRS256SignerAndValidator creates a token signer and validator pair with the given properties from the
// specified RSA private key.
func CreateRS256SignerAndValidator(issuerID string, audience jwt.Audience, privateKeyStore, publicKeyStore KeyGetter, keyID string) (*SignerGetter, Validator) {
	validator := NewRS256Validator(publicKeyStore, issuerID, audience)
	signer := &SignerGetter{
		keyStore: privateKeyStore,
		keyID:    keyID,
	}
	return signer, validator
}

// CreateES256Validator creates a token validator pair with the given properties and jwks public key url
func CreateES256Validator(issuerID string, audience jwt.Audience, publicKeyURL string) (Validator, error) {
	keyStore := NewJWKSGetter(publicKeyURL)
	return NewES256Validator(keyStore, issuerID, audience), nil
}
