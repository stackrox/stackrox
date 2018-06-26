package service

import (
	"sync"

	benchmarkDataStore "bitbucket.org/stack-rox/apollo/central/benchmark/datastore"
	"bitbucket.org/stack-rox/apollo/central/benchmarktrigger/datastore"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), benchmarkDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
