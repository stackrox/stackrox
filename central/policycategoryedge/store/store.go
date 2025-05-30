package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for policy category associations.
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.PolicyCategoryEdge, bool, error)
	Upsert(ctx context.Context, obj *storage.PolicyCategoryEdge) error
	UpsertMany(ctx context.Context, objs []*storage.PolicyCategoryEdge) error
	Delete(ctx context.Context, id string) error
	GetIDs(ctx context.Context) ([]string, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.PolicyCategoryEdge, []int, error)
	DeleteMany(ctx context.Context, ids []string) error
	// Deprecated: use GetByQueryFn instead
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.PolicyCategoryEdge, error)
	GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj *storage.PolicyCategoryEdge) error) error
	DeleteByQuery(ctx context.Context, q *v1.Query) ([]string, error)

	Walk(ctx context.Context, fn func(obj *storage.PolicyCategoryEdge) error) error
}
