// Package jwt parses and validates JSON Web Tokens (JWTs).
package jwt

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// A Validator checks JSON Web Tokens (JWTs) to ensure they are intended for
// this service and are cryptographically trusted.
type Validator interface {
	Validate(rawToken string, jwtClaims *jwt.Claims, extraClaims ...interface{}) error
}

type validator struct {
	keyGetter PublicKeyGetter
	expected  jwt.Expected
	algorithm string
}

// NewRS256Validator validates tokens generated using RS256 (256-bit RSA).
func NewRS256Validator(keys PublicKeyGetter, issuer string, audience jwt.Audience) Validator {
	return validator{
		keyGetter: keys,
		expected:  jwt.Expected{Issuer: issuer, Audience: audience},
		algorithm: string(jose.RS256),
	}
}

// NewES256Validator validates tokens generated using ES256 (ECDSA using P-256 and SHA-256)
func NewES256Validator(keys PublicKeyGetter, issuer string, audience jwt.Audience) Validator {
	return validator{
		keyGetter: keys,
		expected:  jwt.Expected{Issuer: issuer, Audience: audience},
		algorithm: string(jose.ES256),
	}
}

var (
	// ErrNoKeyID means that no key ID was provided, so validation could not be completed.
	ErrNoKeyID = errors.New("no key ID provided")
	// ErrKeyNotFound means that the referenced key was not in the list of known keys.
	ErrKeyNotFound = errors.New("referenced key not found")
	// ErrNoJWTHeaders means that there were no headers in the JWT (and therefore no signatures to verify).
	ErrNoJWTHeaders = errors.New("no headers found in the JWT")
	// ErrInvalidAlgorithm means that the token uses an algorithm not valid for this validator.
	ErrInvalidAlgorithm = errors.New("invalid algorithm used")

	log = logging.LoggerForModule()
)

// Validate validates the token or returns an error.
func (v validator) Validate(rawToken string, claims *jwt.Claims, extraClaims ...interface{}) error {
	token, err := jwt.ParseSigned(rawToken)
	if err != nil {
		return err
	}

	if len(token.Headers) < 1 {
		return ErrNoJWTHeaders
	}

	for _, h := range token.Headers {
		err := v.validateHeaderFields(token, h, claims, extraClaims...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v validator) validateHeaderFields(token *jwt.JSONWebToken, header jose.Header, claims *jwt.Claims, extraClaims ...interface{}) error {
	if header.Algorithm != v.algorithm {
		return ErrInvalidAlgorithm
	}

	if header.KeyID == "" {
		return ErrNoKeyID
	}

	key := v.keyGetter.Key(header.KeyID)
	if key == nil {
		return ErrKeyNotFound
	}

	allClaims := make([]interface{}, len(extraClaims)+1)
	allClaims[0] = claims
	copy(allClaims[1:], extraClaims)
	err := token.Claims(key, allClaims...)
	if err != nil {
		return errors.Wrap(err, "claim processing")
	}

	err = claims.Validate(v.expected.WithTime(time.Now()))
	if err != nil {
		return err
	}
	return nil
}
