package store

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for images.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)

	Get(ctx context.Context, id string) (*storage.Image, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Image, []int, error)

	// GetImageMetadata and GetImageMetadata returns the image without scan/component data.
	GetImageMetadata(ctx context.Context, id string) (*storage.Image, bool, error)
	GetManyImageMetadata(ctx context.Context, id []string) ([]*storage.Image, []int, error)

	Upsert(ctx context.Context, image *storage.Image) error
	Delete(ctx context.Context, id string) error

	UpdateVulnState(ctx context.Context, cve string, imageIDs []string, state storage.VulnerabilityState) error
}
