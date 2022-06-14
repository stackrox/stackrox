package datastore

import (
	"context"

	"github.com/stackrox/rox/central/apitoken/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the gateway to the DB that enforces access control.
type DataStore interface {
	GetTokenOrNil(ctx context.Context, id string) (token *storage.TokenMetadata, err error)
	GetTokens(ctx context.Context, req *v1.GetAPITokensRequest) ([]*storage.TokenMetadata, error)

	AddToken(ctx context.Context, token *storage.TokenMetadata) error
	RevokeToken(ctx context.Context, id string) (exists bool, err error)
}

// New returns a ready-to-use DataStore instance.
func New(storage store.Store) DataStore {
	return &datastoreImpl{storage: storage}
}
