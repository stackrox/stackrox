package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store is the interface to the events data layer.
type Store interface {
	Get(ctx context.Context, id string) (*storage.Event, bool, error)
	Upsert(ctx context.Context, obj *storage.Event) error
	Delete(ctx context.Context, id string) error
	Walk(ctx context.Context, fn func(obj *storage.Event) error) error
}
