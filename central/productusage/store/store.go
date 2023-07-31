package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store interface provides methods to access a persistent storage.
//
//go:generate mockgen-wrapper
type Store interface {
	Upsert(ctx context.Context, obj *storage.Usage) error
	Walk(ctx context.Context, fn func(obj *storage.Usage) error) error
}
