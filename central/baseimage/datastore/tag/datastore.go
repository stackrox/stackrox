package tag

import (
	"context"

	tagStore "github.com/stackrox/rox/central/baseimage/store/tag/postgres"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore provides business logic for base image tag cache operations.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// UpsertMany inserts or updates multiple tags in a batch.
	UpsertMany(ctx context.Context, tags []*storage.BaseImageTag) error

	// DeleteMany removes multiple tags by ID in a batch.
	DeleteMany(ctx context.Context, ids []string) error

	// ListTagsByRepository returns all tags for a repository, ordered by creation
	// timestamp descending (newest first).
	ListTagsByRepository(ctx context.Context, repositoryID string) ([]*storage.BaseImageTag, error)
}

// New returns a new DataStore instance.
func New(store tagStore.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}
