package datastore

import (
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/index"
	"github.com/stackrox/rox/central/cluster/store"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	nodeStore "github.com/stackrox/rox/central/node/globalstore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := store.New(globaldb.GetGlobalDB())
	indexer := index.New(globalindex.GetGlobalIndex())

	var err error
	ad, err = New(storage, indexer, alertDataStore.Singleton(), deploymentDataStore.Singleton(), nodeStore.Singleton(), secretDataStore.Singleton(), connection.ManagerSingleton(), notifierProcessor.Singleton())
	utils.Must(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
