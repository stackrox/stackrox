package datastore

import (
	pgStore "github.com/stackrox/rox/central/cve/image/info/datastore/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())

	ds = New(storage)
}

// Singleton returns a singleton instance of cve time datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
