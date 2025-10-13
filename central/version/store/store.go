package store

import (
	pgStore "github.com/stackrox/rox/central/version/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// A Store stores versions.
type Store interface {
	// GetVersion returns the version found in the DB.
	// If there is no version in the DB, it returns nil and no error, so
	// the caller MUST always check for a nil return value.
	GetVersion() (*storage.Version, error)
	// GetPreviousVersion returns the version found in central_previous.
	// TODO(ROX-18005) -- remove this.  During transition away from serialized version, UpgradeStatus will make this call against
	// the older database.  In that case we will need to process the serialized data.
	GetPreviousVersion() (*storage.Version, error)
	UpdateVersion(*storage.Version) error
}

// NewPostgres returns a new postgres-based version store
func NewPostgres(pg postgres.DB) Store {
	return &storeImpl{pgStore: pgStore.New(pg)}
}
