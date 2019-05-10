package store

import (
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as DataStore
)

func initialize() {
	as = New(store.Singleton(), undostore.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
