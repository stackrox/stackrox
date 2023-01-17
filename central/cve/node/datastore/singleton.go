package datastore

import (
	"github.com/stackrox/rox/central/cve/node/datastore/search"
	"github.com/stackrox/rox/central/cve/node/datastore/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/node/datastore/keyfence"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := postgres.New(globaldb.GetPostgres())
	indexer := postgres.NewIndexer(globaldb.GetPostgres())

	var err error
	ds, err = New(storage, indexer, search.New(storage, indexer), keyfence.NodeKeyFenceSingleton())
	utils.CrashOnError(err)
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
