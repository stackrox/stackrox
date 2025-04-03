package clusterhealth

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for cluster health store.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ClusterHealthStatus, bool, error)
	Upsert(ctx context.Context, obj *storage.ClusterHealthStatus) error
	UpsertMany(ctx context.Context, objs []*storage.ClusterHealthStatus) error
	Delete(ctx context.Context, id ...string) error
	GetIDs(ctx context.Context) ([]string, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ClusterHealthStatus, []int, error)

	Walk(ctx context.Context, fn func(obj *storage.ClusterHealthStatus) error) error
}
