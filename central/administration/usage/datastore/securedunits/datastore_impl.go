package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/administration/usage/store"
	"github.com/stackrox/rox/central/administration/usage/store/cache"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	usageSAC = sac.ForResource(resources.Administration)
)

const (
	page = 1000
)

type dataStoreImpl struct {
	store     store.Store
	clusterDS clusterDataStore
	cache     cache.Cache
}

var _ DataStore = (*dataStoreImpl)(nil)

// Walk calls fn on every record found in the storage. Stops iterating if
// fn returns an error, and returns this error.
func (ds *dataStoreImpl) Walk(ctx context.Context, from time.Time, to time.Time, fn func(*storage.SecuredUnits) error) error {
	if err := sac.VerifyAuthzOK(usageSAC.ReadAllowed(ctx)); err != nil {
		return err
	}
	if from.IsZero() {
		from = time.Unix(0, 0)
	}
	if to.IsZero() {
		to = time.Now()
	}

	pagination := search.NewPagination().
		AddSortOption(search.NewSortOption(search.AdministrationUsageTimestamp)).Limit(page)

	query := search.NewQueryBuilder().AddTimeRangeField(
		search.AdministrationUsageTimestamp, from, to).WithPagination(pagination).ProtoQuery()

	for offset := 0; ; offset += page {
		pagination.Offset(int32(offset))
		units, err := ds.store.GetByQuery(ctx, query)
		if err != nil {
			return errors.Wrap(err, "failed to walk through usage data")
		}
		for _, u := range units {
			if err := fn(u); err != nil {
				return errors.Wrap(err, "error while processing usage data")
			}
		}
		if len(units) < page {
			break
		}
	}
	return nil
}

// GetMaxNodes returns the record with the maximum value of NumNodes.
func (ds *dataStoreImpl) GetMaxNumNodes(ctx context.Context, from time.Time, to time.Time) (*storage.SecuredUnits, error) {
	return ds.getMax(ctx, search.AdministrationUsageNodes, from, to)
}

// GetMaxNumCPUUnits returns the record with the maximum value of NumCpuUnits.
func (ds *dataStoreImpl) GetMaxNumCPUUnits(ctx context.Context, from time.Time, to time.Time) (*storage.SecuredUnits, error) {
	return ds.getMax(ctx, search.AdministrationUsageCPUUnits, from, to)
}

func (ds *dataStoreImpl) getMax(ctx context.Context, label search.FieldLabel, from time.Time, to time.Time) (*storage.SecuredUnits, error) {
	if err := sac.VerifyAuthzOK(usageSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}
	if from.IsZero() {
		from = time.Unix(0, 0)
	}
	if to.IsZero() {
		to = time.Now()
	}

	pagination := search.NewPagination().
		AddSortOption(search.NewSortOption(label).Reversed(true)).Limit(1)

	query := search.NewQueryBuilder().AddTimeRangeField(
		search.AdministrationUsageTimestamp, from, to).WithPagination(pagination).ProtoQuery()

	units, err := ds.store.GetByQuery(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get maximum of usage data")
	}
	if len(units) == 0 {
		return nil, nil
	}
	return units[0], nil
}

// Add saves the current state of an object in storage.
func (ds *dataStoreImpl) Add(ctx context.Context, obj *storage.SecuredUnits) error {
	if err := sac.VerifyAuthzOK(usageSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if obj.Id == "" {
		obj.Id = uuid.NewV4().String()
	}
	return errors.Wrap(ds.store.Upsert(ctx, obj), "failed to upsert usage record")
}

// GetCurrent returns the current usage.
func (ds *dataStoreImpl) GetCurrentUsage(ctx context.Context) (*storage.SecuredUnits, error) {
	if err := sac.VerifyAuthzOK(usageSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}
	ids, err := getClusterIDs(ctx, ds.clusterDS)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster IDs for current usage")
	}
	// Cleanup inactive clusters, doesn't need to be transactional.
	ds.cache.Cleanup(ids)
	return ds.cache.GetCurrent(), nil
}

// AggregateAndReset returns collected metrics for the known clusters. Resets the cache
// for the next iteration.
func (ds *dataStoreImpl) AggregateAndReset(ctx context.Context) (*storage.SecuredUnits, error) {
	if err := sac.VerifyAuthzOK(usageSAC.WriteAllowed(ctx)); err != nil {
		return nil, err
	}
	ids, err := getClusterIDs(ctx, ds.clusterDS)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster IDs for usage snapshot")
	}
	// Cleanup inactive clusters, doesn't need to be transactional.
	ds.cache.Cleanup(ids)
	return ds.cache.AggregateAndReset(), nil
}

// UpdateUsage updates the cache with the metrics of the clusterID cluster.
func (ds *dataStoreImpl) UpdateUsage(ctx context.Context, clusterID string, cm *storage.SecuredUnits) error {
	if err := sac.VerifyAuthzOK(usageSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	ds.cache.UpdateUsage(clusterID, cm)
	return nil
}
