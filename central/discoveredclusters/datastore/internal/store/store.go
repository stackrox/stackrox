package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store is the interface to the discovered cluster data layer.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)

	Get(ctx context.Context, id string) (*storage.DiscoveredCluster, bool, error)
	// Deprecated: use GetByQueryFn instead
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.DiscoveredCluster, error)
	GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj *storage.DiscoveredCluster) error) error
	UpsertMany(ctx context.Context, objs []*storage.DiscoveredCluster) error
	DeleteByQuery(ctx context.Context, query *v1.Query) ([]string, error)
}
