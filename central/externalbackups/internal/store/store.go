package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store implements a store of all external backups in a cluster.
//
//go:generate mockgen-wrapper
type Store interface {
	GetAll(ctx context.Context) ([]*storage.ExternalBackup, error)
	Get(ctx context.Context, id string) (*storage.ExternalBackup, bool, error)
	Upsert(ctx context.Context, backup *storage.ExternalBackup) error
	Delete(ctx context.Context, id string) error
}
