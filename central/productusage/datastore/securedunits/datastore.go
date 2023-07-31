package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/productusage/store"
	"github.com/stackrox/rox/central/productusage/store/cache"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the datastore for usage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// Persistent storage access:

	// Get returns the channel, from which the metrics could be read.
	Get(ctx context.Context, from *types.Timestamp, to *types.Timestamp) (<-chan *storage.SecuredUnits, error)
	// Insert puts metrics to the persistent storage.
	Insert(ctx context.Context, metrics *storage.SecuredUnits) error

	// In-memory storage access:

	// AggregateAndFlush returns the aggregated metrics from the
	// in-memory storage and resets the storage.
	AggregateAndFlush(ctx context.Context) (*storage.SecuredUnits, error)
	// GetCurrentUsage returns the currently known usage.
	GetCurrentUsage(ctx context.Context) (*storage.SecuredUnits, error)
	// UpdateUsage updates the in-memory storage with the cluster metrics.
	UpdateUsage(ctx context.Context, clusterID string, metrics *storage.SecuredUnits) error
}

// New initializes a datastore implementation instance.
func New(store store.Store, clusterStore clusterStoreI) DataStore {
	return &dataStoreImpl{
		store:        store,
		clusterStore: clusterStore,
		cache:        cache.NewCache(),
	}
}
