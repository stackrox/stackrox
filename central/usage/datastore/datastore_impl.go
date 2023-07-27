package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

var (
	previousMetrics = &storage.Usage{}
)

type dataStoreImpl struct {
	cache *cache
}

var _ DataStore = (*dataStoreImpl)(nil)

// Get returns the object, if it exists from the store.
func (ds *dataStoreImpl) Get(ctx context.Context, from *types.Timestamp, to *types.Timestamp) ([]*storage.Usage, error) {
	return nil, errors.New("not implemented")
}

// Upsert saves the current state of an object in storage.
func (ds *dataStoreImpl) Insert(ctx context.Context, metrics *storage.Usage) error {
	return errors.New("not implemented")
}

// GetCurrent returns the current usage.
func (ds *dataStoreImpl) GetCurrent(ctx context.Context) (*storage.Usage, error) {
	ids, err := getClusterIDs(ctx)
	if err != nil {
		log.Debug("Failed to get cluster IDs for current usage: ", err)
		return nil, err
	}

	m := ds.cache.FilterCurrent(ids)
	return m, nil
}

func (ds *dataStoreImpl) CutMetrics(ctx context.Context) (*storage.Usage, error) {
	ids, err := getClusterIDs(ctx)
	if err != nil {
		log.Debug("Failed to get cluster IDs for usage snapshot: ", err)
		return nil, err
	}
	return ds.cache.CutMetrics(ids), nil
}

func (ds *dataStoreImpl) UpdateUsage(clusterID string, cm *central.ClusterMetrics) error {
	ds.cache.UpdateUsage(clusterID, cm)
	return nil
}
