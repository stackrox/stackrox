package datastore

import (
	"github.com/stackrox/rox/central/serviceidentities/internal/store"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ds   DataStore
	once sync.Once
)

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(func() {
		ds = New(store.Singleton())
	})
	return ds
}
