package tokenbasedsource

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// TokenStore is the data source that can list existing tokens.
//
//go:generate mockgen-wrapper
type TokenStore interface {
	GetTokens(ctx context.Context, req *v1.GetAPITokensRequest) ([]*storage.TokenMetadata, error)
}
