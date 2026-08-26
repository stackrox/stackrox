package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store is the interface for AI integration persistence.
type Store interface {
	Get(ctx context.Context, id string) (*storage.AiIntegration, bool, error)
	Upsert(ctx context.Context, obj *storage.AiIntegration) error
	Delete(ctx context.Context, id string) error
	Walk(ctx context.Context, fn func(obj *storage.AiIntegration) error) error
	Exists(ctx context.Context, id string) (bool, error)
}
