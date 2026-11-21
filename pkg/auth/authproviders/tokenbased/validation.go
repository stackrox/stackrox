package tokenbased

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

// httpScopePolicyLayer requires or rejects HTTP scopes according to the source's
// policy. Scope format is validated by the token issuer and validator before
// invoking source layers.
type httpScopePolicyLayer struct {
	sourceID          string
	requireHTTPScopes bool
}

// NewHTTPScopedValidationLayer requires HTTP-scoped claims with sourceID as their
// sole audience. Use a distinct source ID that older Centrals do not recognize,
// so they cannot accept these tokens without enforcing their HTTP restrictions.
func NewHTTPScopedValidationLayer(sourceID string) tokens.SourceLayer {
	return &httpScopePolicyLayer{sourceID: sourceID, requireHTTPScopes: true}
}

func (l *httpScopePolicyLayer) Validate(_ context.Context, claims *tokens.Claims) error {
	if claims == nil {
		return nil
	}
	hasHTTPScopes := claims.HTTPRequestScopes != nil
	for _, role := range claims.InternalRoles {
		if role != nil && role.HTTPRequestScopes != nil {
			hasHTTPScopes = true
			break
		}
	}
	if l.requireHTTPScopes != hasHTTPScopes {
		return errors.New("token HTTP scopes do not match source")
	}
	if l.requireHTTPScopes && (len(claims.Audience) != 1 || claims.Audience[0] != l.sourceID) {
		return errors.New("HTTP-scoped token must have exactly one source audience")
	}
	return nil
}
