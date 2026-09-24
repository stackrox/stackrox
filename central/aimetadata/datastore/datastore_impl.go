package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/aimetadata/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	store store.Store
}

func (ds *datastoreImpl) GetAIMetadata(ctx context.Context, id string) (*storage.AIMetadata, bool, error) {
	return ds.store.Get(ctx, id)
}

func (ds *datastoreImpl) ListAIMetadata(ctx context.Context, query *v1.Query) ([]*storage.AIMetadata, error) {
	return ds.store.GetByQuery(ctx, query)
}

func (ds *datastoreImpl) Search(ctx context.Context, query *v1.Query) ([]search.Result, error) {
	return ds.store.Search(ctx, query)
}

func (ds *datastoreImpl) UpsertAIMetadata(ctx context.Context, metadata *storage.AIMetadata) error {
	if metadata == nil || metadata.GetId() == "" {
		return errors.New("AI metadata id must be defined")
	}
	return ds.store.Upsert(ctx, metadata)
}

func (ds *datastoreImpl) DeleteAIMetadata(ctx context.Context, id string) error {
	return ds.store.Delete(ctx, id)
}
