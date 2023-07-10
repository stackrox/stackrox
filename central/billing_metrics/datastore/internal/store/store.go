package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store stores and retrieves values from the storage.
type Store interface {
	Get(ctx context.Context, metric string) (*storage.Maximus, bool, error)
	Upsert(ctx context.Context, obj *storage.Maximus) error
	Delete(ctx context.Context, metric string) error
}
