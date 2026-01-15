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

	// UpsertImages upserts multiple BaseImage objects and their associated layer digests.
	// Images are processed in chunks to avoid oversized requests.
	// If a chunk fails, earlier chunks remain committed.
	// No retry logic for failed chunks.
	UpsertImages(ctx context.Context, imagesWithLayers map[*storage.BaseImage][]string) error

	GetBaseImage(ctx context.Context, manifestDigest string) (*storage.BaseImage, bool, error)

	// ListCandidateBaseImages returns all base images and their layers whose first layer matches the specified digest.
	// Only the first-layer digest is used for matching.
	// Returns empty slice if no base images matched.
	// Returns error only for system failures (database connection, etc.).
	ListCandidateBaseImages(ctx context.Context, firstLayer string) ([]*storage.BaseImage, error)

	// ListByRepository returns all base images for a specific repository.
	// Returns empty slice if no base images exist for the repository.
	// Returns error only for system failures (database connection, etc.).
	ListByRepository(ctx context.Context, repositoryID string) ([]*storage.BaseImage, error)

	// DeleteMany removes multiple base images by ID in a batch.
	DeleteMany(ctx context.Context, ids []string) error

	// ReplaceByRepository atomically replaces all base images for a repository.
	// Images in the provided map are upserted; existing images not in the map are deleted.
	// This operation is transactional - either all changes succeed or none do.
	ReplaceByRepository(ctx context.Context, repositoryID string, images map[*storage.BaseImage][]string) error
}
