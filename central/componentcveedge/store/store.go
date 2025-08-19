package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for component-cve edges.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Exists(ctx context.Context, id string) (bool, error)

	Get(ctx context.Context, id string) (*storage.ComponentCVEEdge, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ComponentCVEEdge, []int, error)
	GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj *storage.ComponentCVEEdge) error) error
	WalkByQuery(ctx context.Context, query *v1.Query, fn func(obj *storage.ComponentCVEEdge) error) error
}
