package datastore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/nodecomponent/datastore/search"
	"github.com/stackrox/stackrox/central/nodecomponent/datastore/store/postgres"
	"github.com/stackrox/stackrox/central/ranking"
	riskDataStore "github.com/stackrox/stackrox/central/risk/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := postgres.New(globaldb.GetPostgres())
	indexer := postgres.NewIndexer(globaldb.GetPostgres())

	var err error
	ds, err = New(storage, indexer, search.New(storage, indexer), riskDataStore.Singleton(), ranking.NodeComponentRanker())
	utils.CrashOnError(err)
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
