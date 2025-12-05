package datastore

import (
	"context"

	"github.com/stackrox/rox/central/cve/info/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage store.Store
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return ds.storage.Search(ctx, q)
}

func (ds *datastoreImpl) SearchRawImageCVEInfos(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEInfo, error) {
	times := make([]*storage.ImageCVEInfo, 0)
	err := ds.storage.GetByQueryFn(ctx, q, func(cve *storage.ImageCVEInfo) error {
		times = append(times, cve)
		return nil
	})
	return times, err
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	return ds.storage.Exists(ctx, id)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ImageCVEInfo, bool, error) {
	return ds.storage.Get(ctx, id)
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.storage.Count(ctx, q)
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) (times []*storage.ImageCVEInfo, err error) {
	times, _, err = ds.storage.GetMany(ctx, ids)
	return
}

func (ds *datastoreImpl) Upsert(ctx context.Context, time *storage.ImageCVEInfo) error {
	return ds.storage.Upsert(ctx, time)
}

func (ds *datastoreImpl) UpsertMany(ctx context.Context, times []*storage.ImageCVEInfo) error {
	return ds.storage.UpsertMany(ctx, times)
}
