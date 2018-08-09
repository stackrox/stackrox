package service

import (
	"sync"

	benchmarkDataStore "github.com/stackrox/rox/central/benchmark/datastore"
	"github.com/stackrox/rox/central/benchmarkscan/store"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/globaldb"
)

var (
	once sync.Once

	storage store.Store
	as      Service
)

func initialize() {
	storage = store.New(globaldb.GetGlobalDB())
	as = New(storage, benchmarkDataStore.Singleton(), clusterDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
