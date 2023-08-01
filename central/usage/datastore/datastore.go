package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/usage/source"
	"github.com/stackrox/rox/central/usage/store/cache"
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
	UpdateUsage(ctx context.Context, clusterID string, metrics source.UsageSource) error
}

// New initializes a datastore implementation instance.
func New(_ any, clustore clustore) DataStore {
	return &dataStoreImpl{
		clustore: clustore,
		cache:    cache.NewCache(),
	}
}
