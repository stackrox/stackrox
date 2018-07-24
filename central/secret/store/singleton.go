package store

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/globaldb"
)

var (
	once sync.Once

	storage Store
)

func initialize() {
	storage = New(globaldb.GetGlobalDB())
}

// Singleton provides the instance of the Store interface to register.
func Singleton() Store {
	once.Do(initialize)
	return storage
}
