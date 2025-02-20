//go:build sql_integration

package datastore

import (
	"context"
	"testing"
	"time"

	pgStore "github.com/stackrox/rox/central/apitoken/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/apitoken/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGather(t *testing.T) {
	pool := pgtest.ForT(t)
	ds := &datastoreImpl{storage: pgStore.New(pool.DB)}
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	now := time.Now()
	notExpired := now.Add(10 * time.Hour)
	isExpired := now.Add(-10 * time.Hour)

	testCases := map[string]struct {
		tokens                []*storage.TokenMetadata
		expectedTotalTokens   int
		expectedExpiredTokens int
		expectedRevokedTokens int
		expectedValidTokens   int
	}{
		"one valid token": {
			tokens: []*storage.TokenMetadata{
				testutils.GenerateToken(t, &now, &notExpired, false),
			},
			expectedTotalTokens: 1,
			expectedValidTokens: 1,
		},
		"one expired token": {
			tokens: []*storage.TokenMetadata{
				testutils.GenerateToken(t, &now, &isExpired, false),
			},
			expectedTotalTokens:   1,
			expectedExpiredTokens: 1,
		},
		"one revoked token": {
			tokens: []*storage.TokenMetadata{
				testutils.GenerateToken(t, &now, &notExpired, true),
			},
			expectedTotalTokens:   1,
			expectedRevokedTokens: 1,
		},
		"one expired and revoked token": {
			tokens: []*storage.TokenMetadata{
				testutils.GenerateToken(t, &now, &isExpired, true),
			},
			expectedTotalTokens:   1,
			expectedExpiredTokens: 1,
			expectedRevokedTokens: 1,
		},
		"one valid token, one revoked token": {
			tokens: []*storage.TokenMetadata{
				testutils.GenerateToken(t, &now, &notExpired, false),
				testutils.GenerateToken(t, &now, &notExpired, true),
			},
			expectedTotalTokens:   2,
			expectedRevokedTokens: 1,
			expectedValidTokens:   1,
		},
		"one valid token, one expired token": {
			tokens: []*storage.TokenMetadata{
				testutils.GenerateToken(t, &now, &notExpired, false),
				testutils.GenerateToken(t, &now, &isExpired, false),
			},
			expectedTotalTokens:   2,
			expectedExpiredTokens: 1,
			expectedValidTokens:   1,
		},
		"one valid token, one revoked token, one expired token": {
			tokens: []*storage.TokenMetadata{
				testutils.GenerateToken(t, &now, &notExpired, false),
				testutils.GenerateToken(t, &now, &notExpired, true),
				testutils.GenerateToken(t, &now, &isExpired, false),
			},
			expectedTotalTokens:   3,
			expectedExpiredTokens: 1,
			expectedRevokedTokens: 1,
			expectedValidTokens:   1,
		},
		"no tokens": {},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			for _, token := range tc.tokens {
				err := ds.AddToken(ctx, token)
				require.NoError(t, err)
			}

			props, err := Gather(ds)(ctx)
			require.NoError(t, err)

			expectedProps := map[string]any{
				"Total API Tokens":         tc.expectedTotalTokens,
				"Total API Tokens Expired": tc.expectedExpiredTokens,
				"Total API Tokens Revoked": tc.expectedRevokedTokens,
				"Total API Tokens Valid":   tc.expectedValidTokens,
			}
			assert.Equal(t, expectedProps, props)

			for _, token := range tc.tokens {
				err := ds.storage.Delete(ctx, token.GetId())
				require.NoError(t, err)
			}
		})
	}
}
