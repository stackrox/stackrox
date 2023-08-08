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
	store        store.Store
	clusterStore clusterStoreI
	cache        cache.Cache
}

var _ DataStore = (*dataStoreImpl)(nil)

type DataImpl storage.SecuredUnits

func (c *DataImpl) GetTimestamp() *types.Timestamp {
	return (*storage.SecuredUnits)(c).GetTimestamp()
}

func (c *DataImpl) GetNumNodes() int64 {
	return (*storage.SecuredUnits)(c).GetNumNodes()
}

func (c *DataImpl) GetNumCPUUnits() int64 {
	return (*storage.SecuredUnits)(c).GetNumCpuUnits()
}

// Get returns the object, if it exists from the store.
func (ds *dataStoreImpl) Get(ctx context.Context, from *types.Timestamp, to *types.Timestamp) (<-chan Data, error) {
	if err := sac.VerifyAuthzOK(usageSAC.ReadAllowed(ctx)); err != nil {
		return nil, errors.Wrap(err, "cannot permit to get usage data")
	}
	if from == nil {
		from, _ = types.TimestampProto(time.Time{})
	}
	if to == nil {
		to = types.TimestampNow()
	}
	result := make(chan Data)
	go func() {
		defer close(result)
		if err := ds.store.Walk(ctx, func(record *storage.SecuredUnits) error {
			if record.Timestamp.Compare(from) >= 0 && record.Timestamp.Compare(to) < 0 {
				result <- (*DataImpl)(record)
			}
			return nil
		}); err != nil {
			log.Info("Error while walking the product usage table:", err)
		}
	}()
	return result, nil
}

// Insert saves the current state of an object in storage.
func (ds *dataStoreImpl) Insert(ctx context.Context, obj Data) error {
	if err := sac.VerifyAuthzOK(usageSAC.WriteAllowed(ctx)); err != nil {
		return errors.Wrap(err, "cannot permit to insert usage data")
	}
	units := storage.SecuredUnits{
		Id:          uuid.NewV4().String(),
		Timestamp:   obj.GetTimestamp(),
		NumNodes:    obj.GetNumNodes(),
		NumCpuUnits: obj.GetNumCPUUnits(),
	}
	return errors.Wrap(ds.store.Upsert(ctx, &units), "failed to upsert usage record")
}

// GetCurrent returns the current usage.
func (ds *dataStoreImpl) GetCurrentUsage(ctx context.Context) (Data, error) {
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
func (ds *dataStoreImpl) AggregateAndFlush(ctx context.Context) (Data, error) {
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
func (ds *dataStoreImpl) UpdateUsage(ctx context.Context, clusterID string, cm Data) error {
	if err := sac.VerifyAuthzOK(usageSAC.WriteAllowed(ctx)); err != nil {
		return errors.Wrap(err, "cannot permit to update usage data cache")
	}
	units := DataImpl{
		Id:          uuid.NewV4().String(),
		Timestamp:   cm.GetTimestamp(),
		NumNodes:    cm.GetNumNodes(),
		NumCpuUnits: cm.GetNumCPUUnits(),
	}
	ds.cache.UpdateUsage(clusterID, &units)
	return nil
}
