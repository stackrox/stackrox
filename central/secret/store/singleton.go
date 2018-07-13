package store

import (
	"sync"

	globnalDB "bitbucket.org/stack-rox/apollo/central/globaldb/singletons"
)

var (
	once sync.Once

	storage Store
)

func initialize() {
	storage = New(globnalDB.GetGlobalDB())
}

// Singleton provides the instance of the Store interface to register.
func Singleton() Store {
	once.Do(initialize)
	return storage
}
