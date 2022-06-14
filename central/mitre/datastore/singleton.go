package datastore

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	store *mitreAttackStoreImpl
)

func initialize() {
	store = newMitreAttackStore()
}

// Singleton provides the singleton instance of the MitreAttackReadOnlyDataStore interface.
func Singleton() MitreAttackReadOnlyDataStore {
	once.Do(initialize)
	return store
}
