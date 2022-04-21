package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store encapsulates the integration health store
type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	GetIDs(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*storage.IntegrationHealth, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.IntegrationHealth, []int, error)
	Upsert(ctx context.Context, obj *storage.IntegrationHealth) error
	UpsertMany(ctx context.Context, objs []*storage.IntegrationHealth) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
	Walk(ctx context.Context, fn func(obj *storage.IntegrationHealth) error) error
	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)
}
