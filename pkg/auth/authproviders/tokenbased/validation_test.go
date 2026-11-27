package tokenbased

import (
	"errors"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stackrox/rox/pkg/auth/tokens"
	tokensMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTokenSourceValidationLayers(t *testing.T) {
	for name, options := range map[string]func(tokens.RevocationLayer) []TokenAuthProviderOption{
		"scope option first": func(revocation tokens.RevocationLayer) []TokenAuthProviderOption {
			return []TokenAuthProviderOption{
				WithValidationLayer(NewHTTPScopedValidationLayer("scoped-source")),
				WithRevocationLayer(revocation),
			}
		},
		"revocation option first": func(revocation tokens.RevocationLayer) []TokenAuthProviderOption {
			return []TokenAuthProviderOption{
				WithRevocationLayer(revocation),
				WithValidationLayer(NewHTTPScopedValidationLayer("scoped-source")),
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			revocation := tokensMocks.NewMockRevocationLayer(gomock.NewController(t))
			source := NewTokenAuthProvider("scoped-source", "test", "test", options(revocation)...)
			claims := &tokens.Claims{
				Claims: jwt.Claims{Audience: jwt.Audience{source.ID()}},
				RoxClaims: tokens.RoxClaims{
					HTTPRequestScopes: tokens.HTTPRequestScopes{"/metrics": {"GET"}},
				},
			}
			revocation.EXPECT().Validate(t.Context(), claims).Return(nil)
			require.NoError(t, source.Validate(t.Context(), claims))

			revoked := errors.New("revoked")
			revocation.EXPECT().Validate(t.Context(), claims).Return(revoked)
			assert.ErrorIs(t, source.Validate(t.Context(), claims), revoked)

			// Policy failures stop validation before reaching the revocation layer.
			require.ErrorContains(t, source.Validate(t.Context(), &tokens.Claims{}),
				"token HTTP scopes do not match source")
			claims.Audience = jwt.Audience{"other-source"}
			require.ErrorContains(t, source.Validate(t.Context(), claims), "exactly one source audience")

			expiry := time.Now().Add(time.Hour)
			revocation.EXPECT().Revoke("token-id", expiry)
			source.Revoke("token-id", expiry)
			revocation.EXPECT().IsRevoked("token-id").Return(true)
			assert.True(t, source.IsRevoked("token-id"))
		})
	}
}

func TestTokenSourceCustomValidationLayer(t *testing.T) {
	ctrl := gomock.NewController(t)
	validation := tokensMocks.NewMockSourceLayer(ctrl)
	revocation := tokensMocks.NewMockRevocationLayer(ctrl)
	source := NewTokenAuthProvider("test-source", "test", "test",
		WithValidationLayer(validation), WithRevocationLayer(revocation))
	claims := &tokens.Claims{}

	rejected := errors.New("rejected by custom policy")
	validation.EXPECT().Validate(t.Context(), claims).Return(rejected)
	assert.ErrorIs(t, source.Validate(t.Context(), claims), rejected)

	revoked := errors.New("revoked")
	gomock.InOrder(
		validation.EXPECT().Validate(t.Context(), claims).Return(nil),
		revocation.EXPECT().Validate(t.Context(), claims).Return(revoked),
	)
	assert.ErrorIs(t, source.Validate(t.Context(), claims), revoked)
}
