package tokens

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/jwt"
)

// Validator is responsible for validating (and thus parsing) tokens.
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
	srcs, err := v.sources.GetAll(claims.Audience...)
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
