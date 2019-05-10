package datastore

import (
	"context"

	"github.com/stackrox/rox/central/apitoken/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

type datastoreImpl struct {
	storage store.Store
}

func (b *datastoreImpl) AddToken(_ context.Context, token *storage.TokenMetadata) error {
	return b.storage.AddToken(token)
}

func (b *datastoreImpl) GetTokenOrNil(_ context.Context, id string) (token *storage.TokenMetadata, err error) {
	return b.storage.GetTokenOrNil(id)
}

func (b *datastoreImpl) GetTokens(_ context.Context, req *v1.GetAPITokensRequest) (tokens []*storage.TokenMetadata, err error) {
	return b.storage.GetTokens(req)
}

func (b *datastoreImpl) RevokeToken(_ context.Context, id string) (exists bool, err error) {
	return b.storage.RevokeToken(id)
}
