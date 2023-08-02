package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/usage/source"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the datastore for usage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// Persistent storage access:

	// Get returns the channel, from which the metrics could be read.
	Get(ctx context.Context, from *types.Timestamp, to *types.Timestamp) (<-chan *storage.Usage, error)
	// Insert puts metrics to the persistent storage.
	Insert(ctx context.Context, metrics *storage.Usage) error

	// In-memory storage access:

	// AggregateAndFlush returns the aggregated metrics from the
	// in-memory storage and resets the storage.
	AggregateAndFlush(ctx context.Context) (*storage.Usage, error)
	// GetCurrent returns the currently known usage.
	GetCurrent(ctx context.Context) (*storage.Usage, error)
	// UpdateUsage updates the in-memory storage with the cluster metrics.
	UpdateUsage(ctx context.Context, clusterID string, metrics source.UsageSource) error
}

// New initializes a datastore implementation instance.
func New() DataStore {
	return nil
}
