package datastore

import (
	"github.com/stackrox/rox/central/cve/cluster/datastore/search"
	pgStore "github.com/stackrox/rox/central/cve/cluster/datastore/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := pgStore.NewFullStore(globaldb.GetPostgres())
	indexer := pgStore.NewIndexer(globaldb.GetPostgres())

	var err error
	ds, err = New(storage, indexer, search.New(storage, indexer))
	utils.CrashOnError(err)
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
