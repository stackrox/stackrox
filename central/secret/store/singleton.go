package store

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
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
