package backend

import (
	"context"

	"github.com/stackrox/rox/central/apitoken/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

// Backend is the backend for the API tokens component.
type Backend interface {
	GetTokenOrNil(ctx context.Context, tokenID string) (*storage.TokenMetadata, error)
	GetTokens(ctx context.Context, req *v1.GetAPITokensRequest) ([]*storage.TokenMetadata, error)

	IssueRoleToken(ctx context.Context, name string, roleNames []string) (string, *storage.TokenMetadata, error)
	RevokeToken(ctx context.Context, tokenID string) (bool, error)
}

func newBackend(issuer tokens.Issuer, source *sourceImpl, tokenStore datastore.DataStore) Backend {
	return &backendImpl{
		tokenStore: tokenStore,
		source:     source,
		issuer:     issuer,
	}
}
