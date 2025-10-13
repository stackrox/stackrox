package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for network graph configuration.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.NetworkGraphConfig, bool, error)
	Upsert(ctx context.Context, cluster *storage.NetworkGraphConfig) error
}
