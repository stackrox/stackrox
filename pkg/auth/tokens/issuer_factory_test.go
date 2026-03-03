package tokens

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const defaultTTL = 24 * time.Hour

// testSource is a minimal Source implementation for testing.
type testSource struct {
	id string
}

func (s *testSource) ID() string { return s.id }

func (s *testSource) Validate(_ context.Context, _ *Claims) error { return nil }

func createIssuerAndValidator(t *testing.T) (error, Validator, Issuer) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	factory, validator, err := CreateIssuerFactoryAndValidator("test-issuer-expiry", key, "test-expiry-key-id",
		WithDefaultTTL(defaultTTL))
	require.NoError(t, err)

	src := &testSource{id: "test-source-expiry"}
	issuer, err := factory.CreateIssuer(src)
	require.NoError(t, err)
	return err, validator, issuer
}

// Explicitly setting expiry affects `exp` claim in role-based (aka API) tokens.
func TestIssueToken_BasicClaimsAreSet(t *testing.T) {
	err, validator, issuer := createIssuerAndValidator(t)

	roxClaims := RoxClaims{RoleNames: []string{"test-role"}, Name: "token-default-ttl"}
	info, err := issuer.Issue(context.Background(), roxClaims)
	require.NoError(t, err)
	require.NotEmpty(t, info.Token)

	// Round-trip through the validator to parse and verify the token.
	parsed, err := validator.Validate(context.Background(), info.Token)
	require.NoError(t, err)

	// Verify role and name claims survived the round-trip.
	assert.Equal(t, []string{"test-role"}, parsed.RoxClaims.RoleNames)
	assert.Equal(t, "token-2h-lifetime", parsed.RoxClaims.Name)
}

// Explicitly setting expiry affects `exp` claim in role-based (aka API) tokens.
func TestIssueToken_WithExpiry(t *testing.T) {
	err, validator, issuer := createIssuerAndValidator(t)

	expiry := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	roxClaims := RoxClaims{RoleNames: []string{"test-role"}, Name: "token-2h-ttl"}
	info, err := issuer.Issue(context.Background(), roxClaims, WithExpiry(expiry))
	require.NoError(t, err)
	require.NotEmpty(t, info.Token)

	// Round-trip through the validator to parse and verify the token.
	parsed, err := validator.Validate(context.Background(), info.Token)
	require.NoError(t, err)

	// Verify that exp matches the requested expiry.
	assert.Equal(t, expiry.Unix(), parsed.Expiry().Unix(), "exp must match the requested expiry")
}

// Not setting expiry yields tokens with default TTL.
func TestIssueToken_WithoutExpiry_DefaultTTL(t *testing.T) {
	err, validator, issuer := createIssuerAndValidator(t)

	beforeIssue := time.Now()

	roxClaims := RoxClaims{RoleNames: []string{"test-role"}, Name: "token-default-ttl"}
	info, err := issuer.Issue(context.Background(), roxClaims)
	require.NoError(t, err)
	require.NotEmpty(t, info.Token)

	// Round-trip through the validator to parse and verify the token.
	parsed, err := validator.Validate(context.Background(), info.Token)
	require.NoError(t, err)

	// Verify that exp is approximately now + defaultTTL.
	expectedMin := beforeIssue.Add(defaultTTL)
	assert.WithinDuration(t, expectedMin, parsed.Expiry(), 5*time.Second,
		"exp must be approximately now + default TTL")
}
