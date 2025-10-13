package m2m

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
)

// IssuerFromRawIDToken retrieves the issuer from a raw ID token.
// In case the token is malformed (i.e. jwt.ErrTokenMalformed is met), it will return an error.
// Other errors such as an expired token will be ignored.
// Note: This does **not** verify the token's signature or any other claim value.
func IssuerFromRawIDToken(rawIDToken string) (string, error) {
	standardClaims := &jwt.RegisteredClaims{}
	// Explicitly ignore the signature of the ID token for now.
	// This will be handled in a latter part, when the metadata from the provider will be used to verify the signature.
	// This does not pose a security threat, since this is only used to optimize fetching of the correct TokenExchanger.
	// The TokenExchanger will do the final validation of the token including it's signature.
	_, err := jwt.ParseWithClaims(rawIDToken, standardClaims, func(token *jwt.Token) (interface{}, error) {
		return nil, nil
	}, jwt.WithoutClaimsValidation())

	// Deliberately skip errors other than malformed tokens.
	if err != nil && errors.Is(err, jwt.ErrTokenMalformed) {
		return "", errox.InvalidArgs.New("ID token could not be parsed").CausedBy(err)
	}

	if standardClaims.Issuer == "" {
		return "", errox.InvalidArgs.New("empty issuer found in ID token")
	}
	return standardClaims.Issuer, nil
}
