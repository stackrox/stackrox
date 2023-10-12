package datastore

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/administration/usage/store"
	"github.com/stackrox/rox/central/administration/usage/store/cache"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the datastore for administration usage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// Persistent storage access:

	// Walk calls fn on every record found in the storage. Stops iterating if
	// fn returns an error, and returns this error.
	Walk(ctx context.Context, from time.Time, to time.Time, fn func(*storage.SecuredUnits) error) error

	// GetMaxNumNodes returns the record with the maximum value of NumNodes.
	GetMaxNumNodes(ctx context.Context, from time.Time, to time.Time) (*storage.SecuredUnits, error)

	// GetMaxNumCPUUnits returns the record with the maximum value of NumCpuUnits.
	GetMaxNumCPUUnits(ctx context.Context, from time.Time, to time.Time) (*storage.SecuredUnits, error)

	// Add appends metrics to the persistent storage.
	Add(ctx context.Context, metrics *storage.SecuredUnits) error

	//
	// In-memory storage access:
	//
	// With a significant number of secured clusters, if we used the persistent
	// storage as an intermediate location for the collected metrics, the load
	// on the persistent storage may become noticeable. The decision is to use
	// in-memory cache to aggregate metrics and persist it only periodically.

	// AggregateAndReset returns the aggregated metrics from the
	// in-memory storage and resets the storage.
	AggregateAndReset(ctx context.Context) (*storage.SecuredUnits, error)
	// GetCurrentUsage returns the currently known usage.
	GetCurrentUsage(ctx context.Context) (*storage.SecuredUnits, error)
	// UpdateUsage updates the in-memory storage with the cluster metrics.
	UpdateUsage(ctx context.Context, clusterID string, metrics *storage.SecuredUnits) error
}

// New initializes a datastore implementation instance.
func New(store store.Store, clusterDS clusterDataStore) DataStore {
	return &dataStoreImpl{
		store:     store,
		clusterDS: clusterDS,
		cache:     cache.NewCache(),
	}
}
