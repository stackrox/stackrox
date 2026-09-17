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
	Delete(ctx context.Context, id string) error
	GetMany(ctx context.Context, ids []string) ([]*storage.Cluster, []int, error)

	Walk(ctx context.Context, fn func(obj *storage.Cluster) error) error
	WalkByQuery(ctx context.Context, query *v1.Query, fn func(obj *storage.Cluster) error) error

	// GetAllForSAC bypasses SAC filtering to build access scopes. Callers must
	// not modify the returned objects, which may be shared with a store cache.
	GetAllForSAC(ctx context.Context) ([]*storage.Cluster, error)
}
