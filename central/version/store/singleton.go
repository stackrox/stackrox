package store

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
)

var (
	once sync.Once
	as   Store
)

func initialize() {
	as = New(globaldb.GetGlobalDB())
}

// Singleton returns a read-only snapshot of the version store.
// In normal operation of central, this is the singleton that should be used.
func Singleton() ReadOnlyStore {
	once.Do(initialize)
	return as
}

// ReadWriteSingleton returns a Store, which can be updated.
// The version store should ONLY be updated during migrations, not during regular operation of Central.
func ReadWriteSingleton() Store {
	once.Do(initialize)
	return as
}
