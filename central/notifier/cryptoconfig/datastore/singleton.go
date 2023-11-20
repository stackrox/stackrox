package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgstore "github.com/stackrox/rox/central/notifier/cryptoconfig/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := pgstore.New(globaldb.GetPostgres())
	ds = New(storage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
