package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/productusage/store"
	"github.com/stackrox/rox/central/productusage/store/cache"
)

// DataStore is the datastore for usage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// Persistent storage access:

	// Get returns the channel, from which the metrics could be read.
	Get(ctx context.Context, from *types.Timestamp, to *types.Timestamp) (<-chan Data, error)
	// Insert puts metrics to the persistent storage.
	Insert(ctx context.Context, metrics Data) error

	// In-memory storage access:

	// AggregateAndFlush returns the aggregated metrics from the
	// in-memory storage and resets the storage.
	AggregateAndFlush(ctx context.Context) (Data, error)
	// GetCurrentUsage returns the currently known usage.
	GetCurrentUsage(ctx context.Context) (Data, error)
	// UpdateUsage updates the in-memory storage with the cluster metrics.
	UpdateUsage(ctx context.Context, clusterID string, metrics Data) error
}

// Data is the interface to access the stored data values.
type Data interface {
	GetTimestamp() *types.Timestamp
	GetNumNodes() int64
	GetNumCPUUnits() int64
}

// New initializes a datastore implementation instance.
func New(store store.Store, clusterStore clusterStoreI) DataStore {
	return &dataStoreImpl{
		store:        store,
		clusterStore: clusterStore,
		cache:        cache.NewCache(),
	}
}
