package service

import (
	"sync"

	benchmarkDataStore "bitbucket.org/stack-rox/apollo/central/benchmark/datastore"
	"bitbucket.org/stack-rox/apollo/central/benchmarkscan/store"
	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	"bitbucket.org/stack-rox/apollo/central/globaldb"
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
