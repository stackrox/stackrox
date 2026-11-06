package tokens

import (
	"context"
	"slices"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/jwt"
)

// Validator is responsible for validating (and thus parsing) tokens.
//
//go:generate mockgen-wrapper
type Validator interface {
	Validate(ctx context.Context, token string) (*TokenInfo, error)
}

type validator struct {
	validator jwt.Validator
	sources   *sourceStore
}

func newValidator(sources *sourceStore, jwtValidator jwt.Validator) *validator {
	return &validator{
		validator: jwtValidator,
		sources:   sources,
	}
}

func (v *validator) Validate(ctx context.Context, token string) (*TokenInfo, error) {
	var claims Claims
	if err := v.validator.Validate(token, &claims.Claims, &claims.RoxClaims, &claims.Extra); err != nil {
		return nil, err
	}
	sourceIDs := claims.Audience
	perAccess, err := ValidatePerAccessHTTPScopes(&claims.RoxClaims)
	if err != nil {
		return nil, errors.Wrap(err, "invalid per-access HTTP scopes")
	}
	if claims.HTTPRequestScopes != nil || perAccess {
		if len(sourceIDs) != 2 || sourceIDs[1] != scopedTokenAudience {
			return nil, errors.New("HTTP-scoped token is missing its audience marker")
		}
		if err := ValidateHTTPRequestScopes(claims.HTTPRequestScopes); err != nil {
			return nil, errors.Wrap(err, "invalid HTTP request scopes")
		}
		// The marker makes old validators reject the token, but is not itself a token source.
		sourceIDs = sourceIDs[:1]
	} else if slices.Contains(sourceIDs, scopedTokenAudience) {
		return nil, errors.New("unscoped token has an HTTP-scope audience marker")
	}
	srcs, err := v.sources.GetAll(sourceIDs...)
	if err != nil {
		return nil, err
	}
	for _, src := range srcs {
		if err := src.Validate(ctx, &claims); err != nil {
			return nil, errors.Wrap(err, "token rejected by source")
		}
	}
	return &TokenInfo{
		Claims:  &claims,
		Token:   token,
		Sources: srcs,
	}, nil
}
