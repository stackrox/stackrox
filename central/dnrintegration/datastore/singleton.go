package datastore

import (
	"fmt"
	"sync"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/dnrintegration/store"
	"github.com/stackrox/rox/central/globaldb"
)

var (
	once sync.Once

	datastore DataStore
)

func initialize() {
	var err error
	datastore, err = New(store.New(globaldb.GetGlobalDB()), deploymentDataStore.Singleton())
	if err != nil {
		panic(fmt.Sprintf("failed to initialize DNR integration: %s", err))
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return datastore
}
