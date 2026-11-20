package tokenbased

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPScopedTokenSources(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	factory, validator, err := tokens.CreateIssuerFactoryAndValidator("test-issuer", key, "test-key",
		tokens.WithDefaultTTL(time.Hour))
	require.NoError(t, err)
	unscoped := NewTokenAuthProvider("ordinary-tokens", "test", "test")
	// The source's policy is independent of its ID's spelling.
	scoped := NewTokenAuthProvider("restricted-tokens", "test", "test",
		WithValidationLayer(NewHTTPScopedValidationLayer("restricted-tokens")))
	unscopedIssuer, err := factory.CreateIssuer(unscoped)
	require.NoError(t, err)
	scopedIssuer, err := factory.CreateIssuer(scoped)
	require.NoError(t, err)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithHeader("kid", "test-key"))
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		claims tokens.RoxClaims
		scoped bool
	}{
		"unscoped": {},
		"token-wide scopes": {
			claims: tokens.RoxClaims{HTTPRequestScopes: tokens.HTTPRequestScopes{"/metrics": {"GET"}}},
			scoped: true,
		},
		"per-access scopes": {
			claims: tokens.RoxClaims{InternalRoles: []*tokens.InternalRole{
				{HTTPRequestScopes: tokens.HTTPRequestScopes{"/metrics": {"GET"}}},
			}},
			scoped: true,
		},
		"combined scopes": {
			claims: tokens.RoxClaims{
				HTTPRequestScopes: tokens.HTTPRequestScopes{"/v1/": {"GET"}},
				InternalRoles: []*tokens.InternalRole{
					{},
					{HTTPRequestScopes: tokens.HTTPRequestScopes{"/v1/images": {"GET"}, "/v1/blocked/": {}}},
				},
			},
			scoped: true,
		},
		"unscoped first role": {
			claims: tokens.RoxClaims{InternalRoles: []*tokens.InternalRole{
				{},
				{HTTPRequestScopes: tokens.HTTPRequestScopes{"/metrics": {"GET"}}},
			}},
			scoped: true,
		},
		"unscoped last role": {
			claims: tokens.RoxClaims{InternalRoles: []*tokens.InternalRole{
				{HTTPRequestScopes: tokens.HTTPRequestScopes{"/metrics": {"GET"}}},
				{},
			}},
			scoped: true,
		},
		"overlapping role scopes": {
			claims: tokens.RoxClaims{InternalRoles: []*tokens.InternalRole{
				{HTTPRequestScopes: tokens.HTTPRequestScopes{"/metrics": {"GET"}}},
				{HTTPRequestScopes: tokens.HTTPRequestScopes{"/metrics": {"GET"}}},
			}},
			scoped: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			issuer, wrongIssuer := unscopedIssuer, scopedIssuer
			source, wrongSource := unscoped, scoped
			if tc.scoped {
				issuer, wrongIssuer = scopedIssuer, unscopedIssuer
				source, wrongSource = scoped, unscoped
			}
			info, err := issuer.Issue(t.Context(), tc.claims)
			require.NoError(t, err)
			parsed, err := validator.Validate(t.Context(), info.Token)
			require.NoError(t, err)
			assert.Equal(t, jwt.Audience{source.ID()}, parsed.Audience)
			assert.Equal(t, []tokens.Source{source}, parsed.Sources)
			assert.Equal(t, tc.claims, parsed.RoxClaims)

			_, err = wrongIssuer.Issue(t.Context(), tc.claims)
			require.ErrorContains(t, err, "token HTTP scopes do not match source")

			// Sign directly to verify validation independently of issuance checks.
			claims := info.Claims.Claims
			claims.Audience = jwt.Audience{wrongSource.ID()}
			raw, err := jwt.Signed(signer).Claims(claims).Claims(tc.claims).Serialize()
			require.NoError(t, err)
			_, err = validator.Validate(t.Context(), raw)
			require.ErrorContains(t, err, "token HTTP scopes do not match source")
		})
	}

	for name, tc := range map[string]struct {
		audience jwt.Audience
		scopes   tokens.HTTPRequestScopes
		wantErr  string
	}{
		"multiple sources": {
			audience: jwt.Audience{scoped.ID(), scoped.ID()},
			scopes:   tokens.HTTPRequestScopes{"/metrics": {"GET"}},
			wantErr:  "exactly one source audience",
		},
		"invalid scopes": {
			audience: jwt.Audience{scoped.ID()},
			scopes:   tokens.HTTPRequestScopes{"/../metrics": {"GET"}},
			wantErr:  "invalid HTTP request scopes",
		},
	} {
		t.Run(name, func(t *testing.T) {
			claims := jwt.Claims{Issuer: "test-issuer", Audience: tc.audience,
				Expiry: jwt.NewNumericDate(time.Now().Add(time.Hour))}
			raw, err := jwt.Signed(signer).Claims(claims).
				Claims(tokens.RoxClaims{HTTPRequestScopes: tc.scopes}).Serialize()
			require.NoError(t, err)
			_, err = validator.Validate(t.Context(), raw)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
