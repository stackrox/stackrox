package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/authprovider/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/authprovider/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/postgres"
)

// New returns a new Store instance.
func New(storage store.Store) authproviders.Store {
	return &datastoreImpl{
		storage: storage,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) authproviders.Store {
	return &datastoreImpl{
		storage: pgStore.New(pool),
	}
}
