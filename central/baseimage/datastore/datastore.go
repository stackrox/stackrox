package datastore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// DataStore provides access to base images.
//
//go:generate mockgen-wrapper
type DataStore interface {
	UpsertImage(ctx context.Context, image *storage.BaseImage, digests []string) error

	UpsertImages(ctx context.Context, imagesWithLayers map[*storage.BaseImage][]string) error

	GetBaseImage(ctx context.Context, manifestDigest string) (*storage.BaseImage, bool, error)

	// ListCandidateBaseImages returns all base images and their layers whose first layer matches the specified digest.
	// Only the first-layer digest is used for matching.
	// Returns empty slice if no base images matched.
	// Returns error only for system failures (database connection, etc.).
	ListCandidateBaseImages(ctx context.Context, firstLayer string) ([]*storage.BaseImage, error)
}
