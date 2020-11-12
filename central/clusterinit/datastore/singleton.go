package datastore

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	dataStore DataStore
	once      sync.Once
)

// Singleton returns the API token singleton.
func Singleton() DataStore {
	once.Do(func() {
		dataStore = NewInMemory()
	})
	return dataStore
}
