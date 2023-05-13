package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// ContinuousIntegrationStore holds the continuous integration configs.
type ContinuousIntegrationStore interface {
	Get(ctx context.Context, id string) (*storage.ContinuousIntegrationConfig, bool, error)
	Upsert(ctx context.Context, obj *storage.ContinuousIntegrationConfig) error
	Delete(ctx context.Context, id string) error
}
