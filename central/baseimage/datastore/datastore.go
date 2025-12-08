package datastore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// DataStore provides access to base image repositories.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// ListRepositories returns all configured base image repositories.
	// Returns empty slice if no repositories configured.
	// Returns error only for system failures (database connection, etc.).
	ListRepositories(ctx context.Context) ([]*storage.BaseImageRepository, error)
}
