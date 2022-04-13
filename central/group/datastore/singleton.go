package datastore

import (
	"github.com/stackrox/stackrox/central/group/datastore/internal/store"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	ds   DataStore
	once sync.Once
)

// Singleton returns the singleton providing access to the roles store.
func Singleton() DataStore {
	once.Do(func() {
		ds = New(store.Singleton())
	})
	return ds
}
