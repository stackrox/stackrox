package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for alerts.
//
//go:generate mockgen-wrapper
type Store interface {
	Get(ctx context.Context, id string) (*storage.ImageIntegration, bool, error)
	GetAll(ctx context.Context) ([]*storage.ImageIntegration, error)
	Upsert(ctx context.Context, integration *storage.ImageIntegration) error
	Delete(ctx context.Context, id string) error
}
