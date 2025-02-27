package datastore

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/scanaudit/store/postgres"
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
