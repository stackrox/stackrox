package service

import (
	"sync"

	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(processIndicatorDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
