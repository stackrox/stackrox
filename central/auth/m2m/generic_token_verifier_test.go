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
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_genericTokenVerifier_VerifyIDToken(t *testing.T) {
	// Generate RSA key pair for signing.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicKey := &privateKey.PublicKey

	keyID := "test-key-id"

	// Create a mock OIDC server.
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		discovery := map[string]any{
			"issuer":                                server.URL,
			"authorization_endpoint":                server.URL + "/auth",
			"token_endpoint":                        server.URL + "/token",
			"jwks_uri":                              server.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(discovery)
		require.NoError(t, err)
	})

	// JWKS endpoint with real RSA public key.
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		// Convert RSA public key to JWK format
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
		err := json.NewEncoder(w).Encode(jwks)
		require.NoError(t, err)
	})

	// Create real OIDC provider pointing to our mock server.
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, server.URL)
	require.NoError(t, err)

	// Create verifier with real provider.
	verifier := &genericTokenVerifier{provider: provider}

	// Create a valid JWT token.
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   server.URL,
		"sub":   "test-subject",
		"aud":   []string{"test-audience"},
		"exp":   now.Add(time.Hour).Unix(),
		"iat":   now.Unix(),
		"email": "test@example.com",
		"name":  "Test User",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID

	tokenString, err := token.SignedString(privateKey)
	require.NoError(t, err)

	// Test with valid token.
	idToken, err := verifier.VerifyIDToken(ctx, tokenString)

	// Should succeed.
	require.NoError(t, err)
	require.NotNil(t, idToken)
	assert.Equal(t, "test-subject", idToken.Subject)
	assert.Equal(t, []string{"test-audience"}, idToken.Audience)

	// Test claims extraction.
	var extractedClaims map[string]any
	err = idToken.Claims(&extractedClaims)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", extractedClaims["email"])
	assert.Equal(t, "Test User", extractedClaims["name"])

	// Test with invalid token - should fail JWT parsing/verification.
	idToken, err = verifier.VerifyIDToken(ctx, "invalid.jwt.token")
	assert.Nil(t, idToken)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "404")
	assert.NotContains(t, err.Error(), "connection")
}
