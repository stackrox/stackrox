package m2m

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestOIDCServer(t *testing.T) (*httptest.Server, *rsa.PrivateKey, string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicKey := &privateKey.PublicKey
	keyID := "test-key-id"

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		discovery := map[string]any{
			"issuer":                                server.URL,
			"authorization_endpoint":                server.URL + "/auth",
			"token_endpoint":                        server.URL + "/token",
			"jwks_uri":                              server.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(discovery); err != nil {
			t.Errorf("encoding discovery document: %v", err)
			return
		}
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		nBytes := publicKey.N.Bytes()
		eBytes := big.NewInt(int64(publicKey.E)).Bytes()
		jwks := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": keyID,
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(nBytes),
					"e":   base64.RawURLEncoding.EncodeToString(eBytes),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			t.Errorf("encoding JWKS: %v", err)
			return
		}
	})

	return server, privateKey, keyID
}

func signToken(t *testing.T, privateKey *rsa.PrivateKey, keyID string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	tokenString, err := token.SignedString(privateKey)
	require.NoError(t, err)
	return tokenString
}

func Test_genericTokenVerifier_VerifyIDToken(t *testing.T) {
	server, privateKey, keyID := newTestOIDCServer(t)

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, server.URL)
	require.NoError(t, err)

	now := time.Now()
	tokenString := signToken(t, privateKey, keyID, jwt.MapClaims{
		"iss":   server.URL,
		"sub":   "test-subject",
		"aud":   []string{"test-audience"},
		"exp":   now.Add(time.Hour).Unix(),
		"iat":   now.Unix(),
		"email": "test@example.com",
		"name":  "Test User",
	})

	verifier := &genericTokenVerifier{provider: provider}

	idToken, err := verifier.VerifyIDToken(ctx, tokenString)
	require.NoError(t, err)
	require.NotNil(t, idToken)
	assert.Equal(t, "test-subject", idToken.Subject)
	assert.Equal(t, []string{"test-audience"}, idToken.Audience)

	var extractedClaims map[string]any
	err = idToken.Claims(&extractedClaims)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", extractedClaims["email"])
	assert.Equal(t, "Test User", extractedClaims["name"])

	idToken, err = verifier.VerifyIDToken(ctx, "invalid.jwt.token")
	assert.Nil(t, idToken)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "404")
	assert.NotContains(t, err.Error(), "connection")
}

func Test_genericTokenVerifier_AudienceValidation(t *testing.T) {
	server, privateKey, keyID := newTestOIDCServer(t)

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, server.URL)
	require.NoError(t, err)

	now := time.Now()
	baseClaims := func(aud any) jwt.MapClaims {
		claims := jwt.MapClaims{
			"iss": server.URL,
			"sub": "test-subject",
			"exp": now.Add(time.Hour).Unix(),
			"iat": now.Unix(),
		}
		if aud != nil {
			claims["aud"] = aud
		}
		return claims
	}

	testCases := map[string]struct {
		audience      string
		tokenAudience any
		expectError   bool
	}{
		"matching audience passes": {
			audience:      "my-client-id",
			tokenAudience: "my-client-id",
		},
		"matching audience in array passes": {
			audience:      "my-client-id",
			tokenAudience: []string{"other-client", "my-client-id"},
		},
		"non-matching audience is rejected": {
			audience:      "my-client-id",
			tokenAudience: "wrong-audience",
			expectError:   true,
		},
		"non-matching audience array is rejected": {
			audience:      "my-client-id",
			tokenAudience: []string{"wrong-audience", "another-wrong"},
			expectError:   true,
		},
		"empty audience config skips check": {
			audience:      "",
			tokenAudience: "any-audience-value",
		},
		"token without aud claim is rejected when audience is configured": {
			audience:      "my-client-id",
			tokenAudience: nil,
			expectError:   true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			verifier := &genericTokenVerifier{provider: provider, audience: tc.audience}
			tokenString := signToken(t, privateKey, keyID, baseClaims(tc.tokenAudience))

			idToken, err := verifier.VerifyIDToken(ctx, tokenString)
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, idToken)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, idToken)
			}
		})
	}
}
