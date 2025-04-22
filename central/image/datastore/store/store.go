package store

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Store provides storage functionality for images.
//
//go:generate mockgen-wrapper
type Store interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)

	Exists(ctx context.Context, id string) (bool, error)

	Get(ctx context.Context, id string) (*storage.Image, bool, error)
	GetByIDs(ctx context.Context, ids []string) ([]*storage.Image, error)

	// GetImageMetadata and GetImageMetadata returns the image without scan/component data.
	GetImageMetadata(ctx context.Context, id string) (*storage.Image, bool, error)
	GetManyImageMetadata(ctx context.Context, id []string) ([]*storage.Image, error)
	WalkByQuery(ctx context.Context, q *v1.Query, fn func(img *storage.Image) error) error

	Upsert(ctx context.Context, image *storage.Image) error
	Delete(ctx context.Context, id string) error

	UpdateVulnState(ctx context.Context, cve string, imageIDs []string, state storage.VulnerabilityState) error
}
