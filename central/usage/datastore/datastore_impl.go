package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/usage/source"
	"github.com/stackrox/rox/central/usage/store/cache"
	"github.com/stackrox/rox/central/usage/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	previousMetrics = &storage.Usage{}
	usageSAC        = sac.ForResource(resources.Administration)
)

type dataStoreImpl struct {
	clustore clustore
	store    postgres.Store
	cache    cache.Cache
}

var _ DataStore = (*dataStoreImpl)(nil)

// Get returns the object, if it exists from the store.
func (ds *dataStoreImpl) Get(ctx context.Context, from *types.Timestamp, to *types.Timestamp) ([]*storage.Usage, error) {
	if err := sac.VerifyAuthzOK(usageSAC.ReadAllowed(ctx)); err != nil {
		return nil, errors.Wrap(err, "cannot permit to get usage data")
	}
	u, err := ds.store.Get(ctx, from, to)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get usage metrics from store")
	}
	return u, nil
}

// Insert saves the current state of an object in storage.
func (ds *dataStoreImpl) Insert(ctx context.Context, metrics *storage.Usage) error {
	if err := sac.VerifyAuthzOK(usageSAC.WriteAllowed(ctx)); err != nil {
		return errors.Wrap(err, "cannot permit to insert usage data")
	}
	return errors.Wrap(ds.store.Insert(ctx, metrics), "failed to insert metrics to store")
}

// GetCurrent returns the current usage.
func (ds *dataStoreImpl) GetCurrent(ctx context.Context) (*storage.Usage, error) {
	if err := sac.VerifyAuthzOK(usageSAC.ReadAllowed(ctx)); err != nil {
		return nil, errors.Wrap(err, "cannot permit to get current usage data")
	}
	ids, err := getClusterIDs(ctx, ds.clustore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster IDs for current usage")
	}
	return ds.cache.FilterCurrent(ids), nil
}

// CutMetrics returns collected metrics for the known clusters. Resets the cache
// for the next iteration.
func (ds *dataStoreImpl) CutMetrics(ctx context.Context) (*storage.Usage, error) {
	if err := sac.VerifyAuthzOK(usageSAC.WriteAllowed(ctx)); err != nil {
		return nil, errors.Wrap(err, "cannot permit to cut usage data")
	}
	ids, err := getClusterIDs(ctx, ds.clustore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster IDs for usage snapshot")
	}
	return ds.cache.CutMetrics(ids), nil
}

// UpdateUsage updates the cache with the metrics of the clusterID cluster.
func (ds *dataStoreImpl) UpdateUsage(ctx context.Context, clusterID string, cm source.UsageSource) error {
	if err := sac.VerifyAuthzOK(usageSAC.WriteAllowed(ctx)); err != nil {
		return errors.Wrap(err, "cannot permit to update usage data cache")
	}
	ds.cache.UpdateUsage(clusterID, cm)
	return nil
}
