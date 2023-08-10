package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the datastore for usage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// Persistent storage access:

	// Get returns the channel, from which the metrics could be read.
	Walk(ctx context.Context, from *types.Timestamp, to *types.Timestamp, fn func(*storage.SecuredUnits) error) error
	// Upsert puts metrics to the persistent storage.
	Upsert(ctx context.Context, metrics *storage.SecuredUnits) error

	//
	// In-memory storage access:
	//
	// With a significant number of secured clusters, if we used the persistent
	// storage as an intermediate location for the collected metrics, the load
	// on the persistent storage may become noticable. The decision is to use
	// in-memory cache to aggregate metrics and persist it only periodically.

	// AggregateAndFlush returns the aggregated metrics from the
	// in-memory storage and resets the storage.
	AggregateAndFlush(ctx context.Context) (*storage.SecuredUnits, error)
	// GetCurrentUsage returns the currently known usage.
	GetCurrentUsage(ctx context.Context) (*storage.SecuredUnits, error)
	// UpdateUsage updates the in-memory storage with the cluster metrics.
	UpdateUsage(ctx context.Context, clusterID string, metrics *storage.SecuredUnits) error
}

// New initializes a datastore implementation instance.
func New() DataStore {
	return nil
}
