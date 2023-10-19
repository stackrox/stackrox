package datastore

import (
	"github.com/stackrox/rox/central/cve/image/datastore/search"
	pgStore "github.com/stackrox/rox/central/cve/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/image/datastore/keyfence"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())

	ds = New(storage, search.New(storage, pgStore.NewIndexer(globaldb.GetPostgres())), keyfence.ImageKeyFenceSingleton())
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
