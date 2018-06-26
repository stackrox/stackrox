package store

import (
	"sync"

	globaldb "bitbucket.org/stack-rox/apollo/central/globaldb/singletons"
)

var (
	once sync.Once

	storage Store
)

func initialize() {
	storage = New(globaldb.GetGlobalDB())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() Store {
	once.Do(initialize)
	return storage
}
