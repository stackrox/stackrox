package store

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
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
