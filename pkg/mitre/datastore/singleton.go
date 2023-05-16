package datastore

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	store AttackReadOnlyDataStore
)

func initialize() {
	store = NewMitreAttackStore()
}

// Singleton provides the singleton instance of the AttackReadOnlyDataStore interface.
func Singleton() AttackReadOnlyDataStore {
	once.Do(initialize)
	return store
}
