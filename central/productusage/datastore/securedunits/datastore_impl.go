package datastore

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/productusage/store"
	"github.com/stackrox/rox/central/productusage/store/cache"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	previousMetrics = &storage.SecuredUnits{}
	usageSAC        = sac.ForResource(resources.Administration)
)

type dataStoreImpl struct {
	store     store.Store
	clusterDS clusterDataStore
	cache     cache.Cache
}

var _ DataStore = (*dataStoreImpl)(nil)

// Walk returns the object, if it exists from the store.
func (ds *dataStoreImpl) Walk(ctx context.Context, from *types.Timestamp, to *types.Timestamp, fn func(*storage.SecuredUnits) error) error {
	if err := sac.VerifyAuthzOK(usageSAC.ReadAllowed(ctx)); err != nil {
		return errors.Wrap(err, "cannot permit to walk through usage data")
	}
	if from == nil {
		from, _ = types.TimestampProto(time.Time{})
	}
	if to == nil {
		to = types.TimestampNow()
	}
	if err := ds.store.Walk(ctx, func(record *storage.SecuredUnits) error {
		if record.GetTimestamp().Compare(from) >= 0 && record.GetTimestamp().Compare(to) < 0 {
			return fn(record)
		}
		return nil
	}); err != nil {
		log.Info("Error while walking the product usage table:", err)
	}
	return nil
}

// Upsert saves the current state of an object in storage.
func (ds *dataStoreImpl) Upsert(ctx context.Context, obj *storage.SecuredUnits) error {
	if err := sac.VerifyAuthzOK(usageSAC.WriteAllowed(ctx)); err != nil {
		return errors.Wrap(err, "cannot permit to upsert usage data")
	}
	if obj.Id == "" {
		obj.Id = uuid.NewV4().String()
	}
	return errors.Wrap(ds.store.Upsert(ctx, obj), "failed to upsert usage record")
}

// GetCurrent returns the current usage.
func (ds *dataStoreImpl) GetCurrentUsage(ctx context.Context) (*storage.SecuredUnits, error) {
	if err := sac.VerifyAuthzOK(usageSAC.ReadAllowed(ctx)); err != nil {
		return nil, errors.Wrap(err, "cannot permit to get current usage data")
	}
	ids, err := getClusterIDs(ctx, ds.clusterDS)
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
	ids, err := getClusterIDs(ctx, ds.clusterDS)
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
