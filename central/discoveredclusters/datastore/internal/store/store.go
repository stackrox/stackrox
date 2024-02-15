package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store is the interface to the discovered cluster data layer.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.DiscoveredCluster, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.DiscoveredCluster, error)
	UpsertMany(ctx context.Context, objs []*storage.DiscoveredCluster) error
	DeleteByQuery(ctx context.Context, query *v1.Query) ([]string, error)
}
