package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// ClusterStore provides storage functionality for clusters.
//go:generate mockgen-wrapper
type ClusterStore interface {
	Count(ctx context.Context) (int, error)
	Walk(ctx context.Context, fn func(obj *storage.Cluster) error) error

	Get(ctx context.Context, id string) (*storage.Cluster, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Cluster, []int, error)

	Upsert(ctx context.Context, cluster *storage.Cluster) error
	Delete(ctx context.Context, id string) error
}

// ClusterHealthStore provides storage functionality for cluster health.
//go:generate mockgen-wrapper
type ClusterHealthStore interface {
	Get(ctx context.Context, id string) (*storage.ClusterHealthStatus, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ClusterHealthStatus, []int, error)
	Upsert(ctx context.Context, obj *storage.ClusterHealthStatus) error
	UpsertMany(ctx context.Context, objs []*storage.ClusterHealthStatus) error

	Delete(ctx context.Context, id string) error
	Walk(ctx context.Context, fn func(obj *storage.ClusterHealthStatus) error) error
}
