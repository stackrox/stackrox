package repository

import (
	"context"

	repoStore "github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore provides access to base image repositories.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// GetRepository retrieves a base image repository by its ID.
	// Returns the repository, a boolean indicating if it was found, and an error if something went wrong.
	GetRepository(ctx context.Context, id string) (*storage.BaseImageRepository, bool, error)

	// ListRepositories returns all configured base image repositories.
	// Returns empty slice if no repositories configured.
	// Returns error only for system failures (database connection, etc.).
	ListRepositories(ctx context.Context) ([]*storage.BaseImageRepository, error)

	// UpsertRepository inserts or updates the given base image repository.
	// Returns the updated repository and an error, if any.
	UpsertRepository(ctx context.Context, repo *storage.BaseImageRepository) (*storage.BaseImageRepository, error)

	// DeleteRepository removes the base image repository with the specified ID.
	// Returns an error if deletion fails.
	DeleteRepository(ctx context.Context, id string) error
}

// New returns a base image repository DataStore.
func New(s repoStore.Store) DataStore {
	ds := &datastoreImpl{
		store: s,
	}
	return ds
}
