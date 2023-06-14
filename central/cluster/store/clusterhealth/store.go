package clusterhealth

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for cluster health store.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ClusterHealthStatus, bool, error)
	Upsert(ctx context.Context, obj *storage.ClusterHealthStatus) error
	UpsertMany(ctx context.Context, objs []*storage.ClusterHealthStatus) error
	Delete(ctx context.Context, id string) error
	GetIDs(ctx context.Context) ([]string, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ClusterHealthStatus, []int, error)
	DeleteMany(ctx context.Context, ids []string) error

	Walk(ctx context.Context, fn func(obj *storage.ClusterHealthStatus) error) error
}
