package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

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

	Walk(ctx context.Context, fn func(obj *storage.PolicyCategoryEdge) error) error

	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)
}
