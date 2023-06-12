package datastore

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/namespace/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	as DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	indexer := pgStore.NewIndexer(globaldb.GetPostgres())
	var err error
	as, err = New(storage, nil, indexer, deploymentDataStore.Singleton(), ranking.NamespaceRanker())
	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
