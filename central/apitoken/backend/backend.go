package backend

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/apitoken/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

// Backend is the backend for the API tokens component.
type Backend interface {
	GetTokenOrNil(ctx context.Context, tokenID string) (*storage.TokenMetadata, error)
	GetTokens(ctx context.Context, req *v1.GetAPITokensRequest) ([]*storage.TokenMetadata, error)

	IssueRoleToken(ctx context.Context, name string, roleNames []string, expireAt *time.Time) (string, *storage.TokenMetadata, error)
	// IssueEphemeralScopedToken issues a short-lived token with an embedded dynamic access scope.
	// Unlike IssueRoleToken, this method does NOT persist token metadata to the database.
	// The dynamic scope is embedded directly in the token claims.
	// This is intended for short-lived (e.g., 5 minute) tokens used by Sensor's GraphQL gateway.
	IssueEphemeralScopedToken(ctx context.Context, name string, roleNames []string, dynamicScope *storage.DynamicAccessScope, ttl time.Duration) (string, *time.Time, error)
	RevokeToken(ctx context.Context, tokenID string) (bool, error)
}

func newBackend(issuer tokens.Issuer, source *sourceImpl, tokenStore datastore.DataStore) Backend {
	return &backendImpl{
		tokenStore: tokenStore,
		source:     source,
		issuer:     issuer,
	}
}
