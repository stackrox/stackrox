package tokenbasedsource

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/auth/authproviders"
)

// TokenSource is the interface that satisfies the requirements
// for all token sources for token-based identities.
type TokenSource interface {
	authproviders.Provider

	InitFromStore(ctx context.Context, tokenStore TokenStore) error
	Revoke(tokenID string, expiry time.Time)
	IsRevoked(tokenID string) bool
}

// NewTokenSource provides a token validator to support the creation of
// token issuer for token-based identities (API tokens, M2M, OCP plugin).
//
// The output should be used for token issuer source management through
// the API exposed by tokens.IssuerFactory, and could as such be restricted
// to the tokens.Source type. Now, in some cases,
// the token-based authentication flow requires the source to actually
// implement the authproviders.Provider interface.
func NewTokenSource(
	id string,
	name string,
	sourceType string,
	options ...TokenSourceOption,
) TokenSource {
	source := &tokenSourceImpl{
		id:         id,
		name:       name,
		sourceType: sourceType,
	}
	for _, option := range options {
		option(source)
	}
	return source
}
