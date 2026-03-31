package tokens

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stackrox/rox/generated/storage"
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
			InternalRoles: []*InternalRole{internalRole},
			Name:          "token-with-internal-role",
		}
		info, err := issuer.Issue(context.Background(), roxClaims)
		require.NoError(t, err, "Should successfully encode InternalRole to JWT")

		parsed, err := validator.Validate(context.Background(), info.Token)
		require.NoError(t, err)

		require.Len(t, parsed.RoxClaims.InternalRoles, 1)
		assert.Equal(t, "test-internal-role", parsed.RoxClaims.InternalRoles[0].RoleName)
		assert.Len(t, parsed.RoxClaims.InternalRoles[0].Permissions, 2)
	})
}
