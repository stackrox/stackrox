package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides an interface to the underlying data layer
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context) (*storage.Config, bool, error)
	Upsert(ctx context.Context, config *storage.Config) error
}
