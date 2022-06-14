package datastore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/notifier/datastore/internal/store/bolt"
	"github.com/stackrox/stackrox/central/notifier/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
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
