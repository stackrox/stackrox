package service

import (
	"sync"

	benchmarkDataStore "github.com/stackrox/rox/central/benchmark/datastore"
	"github.com/stackrox/rox/central/benchmarkschedule/store"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(store.Singleton(), benchmarkDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
