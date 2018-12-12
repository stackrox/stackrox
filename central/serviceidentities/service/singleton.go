package service

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/serviceidentities/store"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	storage := store.New(globaldb.GetGlobalDB())

	as = New(storage)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
