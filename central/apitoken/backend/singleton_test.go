package backend

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stackrox/rox/central/apitoken/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPITokenSources(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	factory, validator, err := tokens.CreateIssuerFactoryAndValidator("test-issuer", key, "test-key")
	require.NoError(t, err)
	store := &testTokenStore{metadata: make(map[string]*storage.TokenMetadata)}
	backend, err := initBackend(t.Context(), factory, store)
	require.NoError(t, err)
	require.Len(t, store.requests, 1)
	assert.True(t, store.requests[0].GetRevoked())

	issued := make(map[string]*tokens.TokenInfo)
	for name, tc := range map[string]struct {
		scopes   tokens.HTTPRequestScopes
		audience string
	}{
		"unscoped": {audience: id},
		"scoped": {
			scopes:   tokens.HTTPRequestScopes{"/v1/metadata": {"GET"}},
			audience: httpScopedID,
		},
		"deny all": {scopes: tokens.HTTPRequestScopes{"/v1/metadata": {}}, audience: httpScopedID},
	} {
		t.Run(name, func(t *testing.T) {
			token, metadata, err := backend.IssueRoleToken(t.Context(), name, []string{"Analyst"}, nil, tc.scopes)
			require.NoError(t, err)
			info, err := validator.Validate(t.Context(), token)
			require.NoError(t, err)
			assert.Equal(t, []string{tc.audience}, []string(info.Audience))
			assert.Equal(t, tc.scopes, info.HTTPRequestScopes)
			assert.Equal(t, metadata.GetId(), info.ID)
			require.Len(t, info.Sources, 1)
			assert.Equal(t, tc.audience, info.Sources[0].ID())
			issued[name] = info
		})
	}
	require.Len(t, issued, 3)

	for name, info := range issued {
		t.Run("revoke "+name, func(t *testing.T) {
			exists, err := backend.RevokeToken(t.Context(), info.ID)
			require.NoError(t, err)
			require.True(t, exists)
			_, err = validator.Validate(t.Context(), info.Token)
			require.ErrorContains(t, err, "token has been revoked")
			for _, other := range issued {
				source, ok := other.Sources[0].(tokens.RevocationLayer)
				require.True(t, ok)
				assert.True(t, source.IsRevoked(info.ID), "revocation must be shared by both sources")
			}
		})
	}

	// Recreate both sources to verify persisted revocations are loaded into their shared layer.
	restartedFactory, restartedValidator, err := tokens.CreateIssuerFactoryAndValidator("test-issuer", key, "test-key")
	require.NoError(t, err)
	_, err = initBackend(t.Context(), restartedFactory, store)
	require.NoError(t, err)
	require.Len(t, store.requests, 2, "each initialization should query revoked tokens only once")
	assert.True(t, store.requests[1].GetRevoked())
	for _, info := range issued {
		_, err := restartedValidator.Validate(t.Context(), info.Token)
		require.ErrorContains(t, err, "token has been revoked")
	}
}

type testTokenStore struct {
	datastore.DataStore
	metadata map[string]*storage.TokenMetadata
	requests []*v1.GetAPITokensRequest
}

func (s *testTokenStore) AddToken(_ context.Context, metadata *storage.TokenMetadata) error {
	s.metadata[metadata.GetId()] = metadata
	return nil
}

func (s *testTokenStore) GetTokenOrNil(_ context.Context, id string) (*storage.TokenMetadata, error) {
	return s.metadata[id], nil
}

func (s *testTokenStore) GetTokens(_ context.Context, req *v1.GetAPITokensRequest) ([]*storage.TokenMetadata, error) {
	s.requests = append(s.requests, req)
	var result []*storage.TokenMetadata
	for _, metadata := range s.metadata {
		if metadata.GetRevoked() == req.GetRevoked() {
			result = append(result, metadata)
		}
	}
	return result, nil
}

func (s *testTokenStore) RevokeToken(_ context.Context, id string) (bool, error) {
	metadata := s.metadata[id]
	if metadata == nil {
		return false, nil
	}
	metadata.Revoked = true
	return true, nil
}
