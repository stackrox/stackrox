package tokens

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stackrox/rox/generated/storage"
	stackroxjwt "github.com/stackrox/rox/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueTokenJSON(t *testing.T) {
	// Create issuer and validator.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	factory, validator, err := CreateIssuerFactoryAndValidator("test-issuer", key, "test-key-id",
		WithDefaultTTL(defaultTTL))
	require.NoError(t, err)

	src := &testSource{id: "test-source"}
	issuer, err := factory.CreateIssuer(src)
	require.NoError(t, err)

	t.Run("InternalRole with permissions encodes to JWT", func(t *testing.T) {
		internalRole := &InternalRole{
			RoleName: "test-internal-role",
			Permissions: map[storage.Access][]string{
				storage.Access_READ_ACCESS:       {"Deployment"},
				storage.Access_READ_WRITE_ACCESS: {"Image"},
			},
		}
		roxClaims := RoxClaims{
			InternalRoles:     []*InternalRole{internalRole},
			Name:              "token-with-internal-role",
			HTTPRequestScopes: HTTPRequestScopes{"/metrics": {"GET"}},
		}
		info, err := issuer.Issue(context.Background(), roxClaims)
		require.NoError(t, err, "Should successfully encode InternalRole to JWT")
		parts := strings.Split(info.Token, ".")
		require.Len(t, parts, 3)
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		require.NoError(t, err)
		var signedClaims map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(payload, &signedClaims))
		assert.Contains(t, signedClaims, "http")
		assert.NotContains(t, signedClaims, "http_request_scopes")
		var signedHTTP HTTPRequestScopes
		require.NoError(t, json.Unmarshal(signedClaims["http"], &signedHTTP))
		assert.Equal(t, roxClaims.HTTPRequestScopes, signedHTTP)

		parsed, err := validator.Validate(context.Background(), info.Token)
		require.NoError(t, err)

		require.Len(t, parsed.RoxClaims.InternalRoles, 1)
		assert.Equal(t, "test-internal-role", parsed.RoxClaims.InternalRoles[0].RoleName)
		assert.Len(t, parsed.RoxClaims.InternalRoles[0].Permissions, 2)
		assert.Equal(t, roxClaims.HTTPRequestScopes, parsed.RoxClaims.HTTPRequestScopes)
	})
	t.Run("empty non-nil HTTP scopes cannot be signed as unscoped", func(t *testing.T) {
		_, err := issuer.Issue(context.Background(), RoxClaims{HTTPRequestScopes: HTTPRequestScopes{}})
		require.ErrorContains(t, err, "HTTP request scopes must be nil or contain a path")
	})
}

func TestTokenAudienceCompatibility(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	factory, validator, err := CreateIssuerFactoryAndValidator("test-issuer", key, "test-key-id",
		WithDefaultTTL(defaultTTL))
	require.NoError(t, err)
	src := &testSource{id: "test-source"}
	issuer, err := factory.CreateIssuer(src)
	require.NoError(t, err)

	// This mirrors the old validator: verify the JWT, then resolve every audience as a token source.
	oldJWTValidator := stackroxjwt.NewRS256Validator(
		stackroxjwt.NewSingleKeyStore(key.Public(), "test-key-id"), "test-issuer", nil)
	oldSources := newSourceStore()
	require.NoError(t, oldSources.Register(src))

	tests := map[string]struct {
		scopes      HTTPRequestScopes
		audience    jwt.Audience
		oldAccepted bool
	}{
		"unscoped tokens keep their single source audience": {
			audience:    jwt.Audience{src.ID()},
			oldAccepted: true,
		},
		"HTTP-scoped tokens carry an incompatible audience": {
			scopes:   HTTPRequestScopes{"/metrics": {"GET"}},
			audience: jwt.Audience{src.ID(), scopedTokenAudience},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			info, err := issuer.Issue(t.Context(), RoxClaims{HTTPRequestScopes: tc.scopes})
			require.NoError(t, err)
			parsed, err := validator.Validate(t.Context(), info.Token)
			require.NoError(t, err)
			assert.Equal(t, tc.audience, parsed.Audience)
			assert.Equal(t, tc.scopes, parsed.HTTPRequestScopes)
			require.Equal(t, []Source{src}, parsed.Sources)

			var oldClaims jwt.Claims
			require.NoError(t, oldJWTValidator.Validate(info.Token, &oldClaims))
			assert.Equal(t, tc.audience, oldClaims.Audience)
			oldTokenSources, err := oldSources.GetAll(oldClaims.Audience...)
			if tc.oldAccepted {
				require.NoError(t, err)
				assert.Equal(t, []Source{src}, oldTokenSources)
			} else {
				require.ErrorContains(t, err, "invalid source "+scopedTokenAudience)
			}
		})
	}
}

func TestPerAccessTokenAudienceAndValidation(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	factory, validator, err := CreateIssuerFactoryAndValidator("test-issuer", key, "test-key-id", WithDefaultTTL(defaultTTL))
	require.NoError(t, err)
	src := &testSource{id: "test-source"}
	issuer, err := factory.CreateIssuer(src)
	require.NoError(t, err)
	claims := RoxClaims{
		Clusters: ClusterScopes{"cluster": {"namespace"}},
		InternalRoles: []*InternalRole{
			{RoleName: "deployments", Permissions: map[storage.Access][]string{storage.Access_READ_ACCESS: {"Deployment"}},
				HTTPRequestScopes: HTTPRequestScopes{"/v1/deployments": {"GET"}}},
			{RoleName: "images", Permissions: map[storage.Access][]string{storage.Access_READ_ACCESS: {"Image"}},
				HTTPRequestScopes: HTTPRequestScopes{"/v1/images": {"GET"}}},
		},
	}
	info, err := issuer.Issue(t.Context(), claims)
	require.NoError(t, err)
	parts := strings.Split(info.Token, ".")
	require.Len(t, parts, 3)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var signed struct {
		Access []struct {
			Permissions       map[string][]string `json:"permissions"`
			HTTPRequestScopes HTTPRequestScopes    `json:"http"`
		} `json:"access"`
	}
	require.NoError(t, json.Unmarshal(payload, &signed))
	require.Len(t, signed.Access, 2)
	for i, resource := range []string{"Deployment", "Image"} {
		assert.Equal(t, []string{resource}, signed.Access[i].Permissions["read"])
		assert.Equal(t, claims.InternalRoles[i].HTTPRequestScopes, signed.Access[i].HTTPRequestScopes)
	}
	parsed, err := validator.Validate(t.Context(), info.Token)
	require.NoError(t, err)
	assert.Equal(t, jwt.Audience{src.ID(), scopedTokenAudience}, parsed.Audience)
	assert.Nil(t, parsed.HTTPRequestScopes)
	assert.Equal(t, claims.Clusters, parsed.Clusters)
	assert.Equal(t, claims.InternalRoles[0].HTTPRequestScopes, parsed.InternalRoles[0].HTTPRequestScopes)

	withoutMarker := *info.Claims
	withoutMarker.Audience = jwt.Audience{src.ID()}
	badToken, err := factory.(*issuerFactory).encode(&withoutMarker)
	require.NoError(t, err)
	_, err = validator.Validate(t.Context(), badToken)
	require.ErrorContains(t, err, "missing its audience marker")

	claims.HTTPRequestScopes = HTTPRequestScopes{"/": {"GET"}}
	_, err = issuer.Issue(t.Context(), claims)
	require.ErrorContains(t, err, "cannot be combined")
}

func TestTokenValidatorRejectsInvalidAudiences(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	factory, validator, err := CreateIssuerFactoryAndValidator("test-issuer", key, "test-key-id",
		WithDefaultTTL(defaultTTL))
	require.NoError(t, err)
	src := &testSource{id: "test-source"}
	issuer, err := factory.CreateIssuer(src)
	require.NoError(t, err)
	info, err := issuer.Issue(t.Context(), RoxClaims{HTTPRequestScopes: HTTPRequestScopes{"/metrics": {"GET"}}})
	require.NoError(t, err)

	tests := map[string]struct {
		audience jwt.Audience
		scopes   HTTPRequestScopes
		wantErr  string
	}{
		"unknown single source audience": {
			audience: jwt.Audience{"unknown-source"},
			wantErr:  "invalid source unknown-source",
		},
		"unknown source audience": {
			audience: jwt.Audience{"unknown-source", scopedTokenAudience},
			scopes:   HTTPRequestScopes{"/metrics": {"GET"}},
			wantErr:  "invalid source unknown-source",
		},
		"scoped token missing marker": {
			audience: jwt.Audience{src.ID()},
			scopes:   HTTPRequestScopes{"/metrics": {"GET"}},
			wantErr:  "HTTP-scoped token is missing its audience marker",
		},
		"scoped token with marker only": {
			audience: jwt.Audience{scopedTokenAudience},
			scopes:   HTTPRequestScopes{"/metrics": {"GET"}},
			wantErr:  "HTTP-scoped token is missing its audience marker",
		},
		"unscoped token with marker": {
			audience: jwt.Audience{src.ID(), scopedTokenAudience},
			wantErr:  "unscoped token has an HTTP-scope audience marker",
		},
		"scoped token with invalid HTTP path": {
			audience: jwt.Audience{src.ID(), scopedTokenAudience},
			scopes:   HTTPRequestScopes{"/v1/../admin": {"GET"}},
			wantErr:  "invalid HTTP request scopes",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			claims := *info.Claims
			claims.Audience = tc.audience
			claims.HTTPRequestScopes = tc.scopes
			token, err := factory.(*issuerFactory).encode(&claims)
			require.NoError(t, err)
			_, err = validator.Validate(t.Context(), token)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
