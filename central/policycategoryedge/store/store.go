package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for policy category associations.
type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.PolicyCategoryEdge, bool, error)
	Upsert(ctx context.Context, obj *storage.PolicyCategoryEdge) error
	UpsertMany(ctx context.Context, objs []*storage.PolicyCategoryEdge) error
	Delete(ctx context.Context, id string) error
	GetIDs(ctx context.Context) ([]string, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.PolicyCategoryEdge, []int, error)
	DeleteMany(ctx context.Context, ids []string) error
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.PolicyCategoryEdge, error)
	GetAll(ctx context.Context) ([]*storage.PolicyCategoryEdge, error)
	DeleteByQuery(ctx context.Context, q *v1.Query) ([]string, error)

	Walk(ctx context.Context, fn func(obj *storage.PolicyCategoryEdge) error) error
}
