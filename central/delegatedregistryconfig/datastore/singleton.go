package datastore

import (
	pgStore "github.com/stackrox/rox/central/delegatedregistryconfig/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	d DataStore
)

func initialize() {
	d = New(pgStore.New(globaldb.GetPostgres()))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return d
}
