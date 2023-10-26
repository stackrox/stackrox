package datastore

import (
	"testing"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
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
}

// New returns an instance of DataStore.
func New(complianceIntegrationStorage pgStore.Store) DataStore {
	ds := &datastoreImpl{
		storage: complianceIntegrationStorage,
	}
	return ds
}

// NewForTestOnly returns an instance of DataStore only for tests.
func NewForTestOnly(_ *testing.T, complianceIntegrationStorage pgStore.Store) DataStore {
	ds := &datastoreImpl{
		storage: complianceIntegrationStorage,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	store := pgStore.New(pool)
	return New(store), nil
}
