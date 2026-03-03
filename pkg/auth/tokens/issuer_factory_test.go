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

type testSource struct {
	id string
}

func (s *testSource) ID() string { return s.id }

func (s *testSource) Validate(_ context.Context, _ *Claims) error { return nil }

func TestIssueToken(t *testing.T) {
	// Create issuer and validator.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	factory, validator, err := CreateIssuerFactoryAndValidator("test-issuer", key, "test-key-id",
		WithDefaultTTL(defaultTTL))
	require.NoError(t, err)

	src := &testSource{id: "test-source"}
	issuer, err := factory.CreateIssuer(src)
	require.NoError(t, err)

	t.Run("basic claims survive round-trip", func(t *testing.T) {
		roxClaims := RoxClaims{RoleNames: []string{"test-role"}, Name: "test-token"}
		info, err := issuer.Issue(context.Background(), roxClaims)
		require.NoError(t, err)

		parsed, err := validator.Validate(context.Background(), info.Token)
		require.NoError(t, err)

		assert.Equal(t, []string{"test-role"}, parsed.RoxClaims.RoleNames)
		assert.Equal(t, "test-token", parsed.RoxClaims.Name)
	})

	t.Run("explicit expiry sets exp claim in role-based (aka API) tokens", func(t *testing.T) {
		expiry := time.Now().Add(2 * time.Hour).Truncate(time.Second)

		roxClaims := RoxClaims{RoleNames: []string{"test-role"}, Name: "token-2h-ttl"}
		info, err := issuer.Issue(context.Background(), roxClaims, WithExpiry(expiry))
		require.NoError(t, err)

		parsed, err := validator.Validate(context.Background(), info.Token)
		require.NoError(t, err)

		assert.Equal(t, expiry.Unix(), parsed.Expiry().Unix())
	})

	t.Run("omitted expiry falls back to default TTL", func(t *testing.T) {
		beforeIssue := time.Now()

		roxClaims := RoxClaims{RoleNames: []string{"test-role"}, Name: "token-default-ttl"}
		info, err := issuer.Issue(context.Background(), roxClaims)
		require.NoError(t, err)

		parsed, err := validator.Validate(context.Background(), info.Token)
		require.NoError(t, err)

		// Verify that exp is approximately now + defaultTTL.
		expected := beforeIssue.Add(defaultTTL)
		assert.WithinDuration(t, expected, parsed.Expiry(), 5*time.Second,
			"exp must be approximately now + default TTL")
	})
}
