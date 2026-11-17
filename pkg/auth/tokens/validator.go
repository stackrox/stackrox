package tokens

import (
	"context"
	"strings"

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
		if len(sourceIDs) != 1 {
			return nil, errors.New("HTTP-scoped token must have exactly one scoped source audience")
		}
		sourceID, ok := strings.CutSuffix(sourceIDs[0], httpScopedSourceSuffix)
		if !ok || sourceID == "" {
			return nil, errors.New("HTTP-scoped token is missing its scoped source audience")
		}
		if err := ValidateHTTPRequestScopes(claims.HTTPRequestScopes); err != nil {
			return nil, errors.Wrap(err, "invalid HTTP request scopes")
		}
		sourceIDs = []string{sourceID}
	} else {
		for _, sourceID := range sourceIDs {
			if strings.HasSuffix(sourceID, httpScopedSourceSuffix) {
				return nil, errors.New("unscoped token has an HTTP-scoped source audience")
			}
		}
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
