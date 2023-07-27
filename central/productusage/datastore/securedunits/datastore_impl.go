package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/productusage/store/cache"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	previousMetrics = &storage.SecuredUnits{}
	usageSAC        = sac.ForResource(resources.Administration)
)

type dataStoreImpl struct {
	clusterStore clusterStoreI
	cache        cache.Cache
}

var _ DataStore = (*dataStoreImpl)(nil)

// Get returns the object, if it exists from the store.
func (ds *dataStoreImpl) Walk(ctx context.Context, _ *types.Timestamp, _ *types.Timestamp, _ func(m *storage.SecuredUnits) error) error {
	if err := sac.VerifyAuthzOK(usageSAC.ReadAllowed(ctx)); err != nil {
		return errors.Wrap(err, "cannot permit to get usage data")
	}
	return errors.New("not implemented")
}

// Upsert saves the current state of an object in storage.
func (ds *dataStoreImpl) Upsert(ctx context.Context, _ *storage.SecuredUnits) error {
	if err := sac.VerifyAuthzOK(usageSAC.WriteAllowed(ctx)); err != nil {
		return errors.Wrap(err, "cannot permit to upsert usage data")
	}
	return errors.New("not implemented")
}

// GetCurrent returns the current usage.
func (ds *dataStoreImpl) GetCurrentUsage(ctx context.Context) (*storage.SecuredUnits, error) {
	if err := sac.VerifyAuthzOK(usageSAC.ReadAllowed(ctx)); err != nil {
		return nil, errors.Wrap(err, "cannot permit to get current usage data")
	}
	ids, err := getClusterIDs(ctx, ds.clusterStore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster IDs for current usage")
	}
	ds.cache.Cleanup(ids)
	return ds.cache.GetCurrent(), nil
}

// AggregateAndFlush returns collected metrics for the known clusters. Resets the cache
// for the next iteration.
func (ds *dataStoreImpl) AggregateAndFlush(ctx context.Context) (*storage.SecuredUnits, error) {
	if err := sac.VerifyAuthzOK(usageSAC.WriteAllowed(ctx)); err != nil {
		return nil, errors.Wrap(err, "cannot permit to get the aggregate usage data")
	}
	ids, err := getClusterIDs(ctx, ds.clusterStore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster IDs for usage snapshot")
	}
	ds.cache.Cleanup(ids)
	return ds.cache.AggregateAndFlush(), nil
}

// UpdateUsage updates the cache with the metrics of the clusterID cluster.
func (ds *dataStoreImpl) UpdateUsage(ctx context.Context, clusterID string, cm *storage.SecuredUnits) error {
	if err := sac.VerifyAuthzOK(usageSAC.WriteAllowed(ctx)); err != nil {
		return errors.Wrap(err, "cannot permit to update usage data cache")
	}
	ds.cache.UpdateUsage(clusterID, cm)
	return nil
}
