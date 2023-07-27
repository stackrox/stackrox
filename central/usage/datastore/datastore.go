package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the datastore for usage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// Persistent storage
	Get(ctx context.Context, from *types.Timestamp, to *types.Timestamp) ([]*storage.Usage, error)
	Insert(ctx context.Context, metrics *storage.Usage) error

	// In-memory storage
	CutMetrics(ctx context.Context) (*storage.Usage, error)
	GetCurrent(ctx context.Context) (*storage.Usage, error)
	UpdateUsage(clusterID string, metrics *central.ClusterMetrics) error
}

// New initializes a datastore implementation instance.
func New(any) DataStore {
	return nil
}
