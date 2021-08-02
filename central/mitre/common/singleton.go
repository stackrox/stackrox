package common

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

// Singleton provides the singleton instance of the MitreAttackReadOnlyStore interface.
func Singleton() MitreAttackReadOnlyStore {
	once.Do(initialize)
	return store
}

// rwSingleton provides the singleton instance of the mitreAttackStore interface.
func rwSingleton() mitreAttackStore {
	once.Do(initialize)
	return store
}
