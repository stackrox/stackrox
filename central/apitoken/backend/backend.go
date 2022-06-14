package backend

import (
	"context"

	"github.com/stackrox/stackrox/central/apitoken/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/auth/tokens"
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
