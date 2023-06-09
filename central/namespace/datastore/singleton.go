package datastore

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/idmap"
	pgStore "github.com/stackrox/rox/central/namespace/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	as DataStore
)

func initialize() {
	var dackBox *dackbox.DackBox
	storage := pgStore.New(globaldb.GetPostgres())
	indexer := pgStore.NewIndexer(globaldb.GetPostgres())
	var err error
	as, err = New(storage, dackBox, indexer, deploymentDataStore.Singleton(), ranking.NamespaceRanker(), idmap.StorageSingleton())
	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
