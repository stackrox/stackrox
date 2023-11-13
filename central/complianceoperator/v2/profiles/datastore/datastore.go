package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for storing/retrieving compliance operator profiles.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// UpsertProfile adds the profile to the database
	UpsertProfile(ctx context.Context, result *storage.ComplianceOperatorProfileV2) error

	// DeleteProfile removes a profile from the database
	DeleteProfile(ctx context.Context, id string) error
}

// New returns an instance of DataStore.
func New(complianceProfileStorage pgStore.Store) DataStore {
	ds := &datastoreImpl{
		store: complianceProfileStorage,
	}
	return ds
}

// NewForTestOnly returns an instance of DataStore only for tests.
func NewForTestOnly(_ *testing.T, complianceProfileStorage pgStore.Store) DataStore {
	ds := &datastoreImpl{
		store: complianceProfileStorage,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	store := pgStore.New(pool)
	return New(store), nil
}
