package datastore

import (
	"github.com/stackrox/rox/central/externalbackups/internal/store"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ds   DataStore
	once sync.Once
)

// Singleton returns the singleton providing access to the external backups store.
func Singleton() DataStore {
	once.Do(func() {
		ds = New(store.Singleton())
	})
	return ds
}
