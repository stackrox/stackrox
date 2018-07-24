package service

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/globaldb"
	"bitbucket.org/stack-rox/apollo/central/serviceidentities/store"
)

var (
	once sync.Once

	storage store.Store

	as Service
)

func initialize() {
	storage = store.New(globaldb.GetGlobalDB())

	as = New(storage)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
