package datastore

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/namespace/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	as = New(storage, pgStore.NewIndexer(globaldb.GetPostgres()), deploymentDataStore.Singleton(), ranking.NamespaceRanker())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
