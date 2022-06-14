package store

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/networkpolicies/datastore/internal/store/bolt"
	"github.com/stackrox/stackrox/central/networkpolicies/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Store
)

func initialize() {
	if features.PostgresDatastore.Enabled() {
		as = postgres.New(globaldb.GetPostgres())
	} else {
		as = bolt.New(globaldb.GetGlobalDB())
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() Store {
	once.Do(initialize)
	return as
}
