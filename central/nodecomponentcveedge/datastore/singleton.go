package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/nodecomponentcveedge/datastore/search"
	pgStore "github.com/stackrox/rox/central/nodecomponentcveedge/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	indexer := pgStore.NewIndexer(globaldb.GetPostgres())

	ds = New(storage, indexer, search.New(storage, indexer))
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
