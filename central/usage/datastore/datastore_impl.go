package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/usage/source"
	"github.com/stackrox/rox/central/usage/store/cache"
	"github.com/stackrox/rox/generated/storage"
)

var (
	previousMetrics = &storage.Usage{}
)

type dataStoreImpl struct {
	clustore clustore
	cache    cache.Cache
}

var _ DataStore = (*dataStoreImpl)(nil)

// Get returns the object, if it exists from the store.
func (ds *dataStoreImpl) Get(_ context.Context, _ *types.Timestamp, _ *types.Timestamp) ([]*storage.Usage, error) {
	return nil, errors.New("not implemented")
}

// Insert saves the current state of an object in storage.
func (ds *dataStoreImpl) Insert(_ context.Context, _ *storage.Usage) error {
	return errors.New("not implemented")
}

// GetCurrent returns the current usage.
func (ds *dataStoreImpl) GetCurrent(ctx context.Context) (*storage.Usage, error) {
	ids, err := getClusterIDs(ctx, ds.clustore)
	if err != nil {
		return nil, err
	}

	m := ds.cache.FilterCurrent(ids)
	return m, nil
}

// CutMetrics returns collected metrics for the known clusters. Resets the cache
// for the next iteration.
func (ds *dataStoreImpl) CutMetrics(ctx context.Context) (*storage.Usage, error) {
	ids, err := getClusterIDs(ctx, ds.clustore)
	if err != nil {
		return nil, err
	}
	return ds.cache.CutMetrics(ids), nil
}

// UpdateUsage updates the cache with the metrics of the clusterID cluster.
func (ds *dataStoreImpl) UpdateUsage(clusterID string, cm source.UsageSource) {
	ds.cache.UpdateUsage(clusterID, cm)
}
