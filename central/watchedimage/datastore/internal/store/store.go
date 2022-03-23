package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store is a store for watched images.
type Store interface {
	Upsert(ctx context.Context, obj *storage.WatchedImage) error
	Walk(ctx context.Context, fn func(obj *storage.WatchedImage) error) error
	Delete(ctx context.Context, name string) error
	Exists(ctx context.Context, name string) (bool, error)
}
