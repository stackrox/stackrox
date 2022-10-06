package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for resource collections.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ResourceCollection, bool, error)
	GetIDs(ctx context.Context) ([]string, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.ResourceCollection, []int, error)

	Upsert(context.Context, *storage.ResourceCollection) error
	Delete(ctx context.Context, id string) error

	Walk(ctx context.Context, fn func(obj *storage.ResourceCollection) error) error
}
