package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/notifier/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/notifier/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as DataStore
)

func initialize() {
	if features.PostgresDatastore.Enabled() {
		as = New(postgres.New(globaldb.GetPostgres()))
	} else {
		as = New(bolt.New(globaldb.GetGlobalDB()))
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
