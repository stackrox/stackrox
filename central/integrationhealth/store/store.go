package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store encapsulates the integration health store
type Store interface {
	Get(ctx context.Context, id string) (*storage.IntegrationHealth, bool, error)
	Upsert(ctx context.Context, obj *storage.IntegrationHealth) error
	Delete(ctx context.Context, id string) error
	Walk(ctx context.Context, fn func(obj *storage.IntegrationHealth) error) error
}
