package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for clusters.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Get(ctx context.Context, id string) (*storage.Cluster, bool, error)
	Upsert(ctx context.Context, obj *storage.Cluster) error
	Delete(ctx context.Context, id ...string) error
	GetMany(ctx context.Context, ids []string) ([]*storage.Cluster, []int, error)

	Walk(ctx context.Context, fn func(obj *storage.Cluster) error) error
}
