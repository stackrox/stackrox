package bolt

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides the interface to the underlying storage
type Store interface {
	Get(ctx context.Context, id string) (*storage.NetworkPolicy, bool, error)
	Upsert(ctx context.Context, obj *storage.NetworkPolicy) error
	Delete(ctx context.Context, id string) error

	Walk(ctx context.Context, fn func(obj *storage.NetworkPolicy) error) error
}
