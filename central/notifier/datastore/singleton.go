package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/notifier/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as DataStore
)

func initialize() {
	as = New(pgStore.New(globaldb.GetPostgres()))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
