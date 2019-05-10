package datastore

import (
	"github.com/stackrox/rox/central/notifier/datastore/internal/store"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as DataStore
)

func initialize() {
	as = New(store.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
