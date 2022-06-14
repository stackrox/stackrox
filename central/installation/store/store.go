package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides an interface to the underlying data layer
type Store interface {
	Get(ctx context.Context) (*storage.InstallationInfo, bool, error)
	Upsert(ctx context.Context, installationinfo *storage.InstallationInfo) error
}
