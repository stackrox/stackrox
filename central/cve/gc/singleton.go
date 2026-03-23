package gc

import (
	"sync"

	cvev2store "github.com/stackrox/rox/central/cve/image/v2/datastore/store"
)

var (
	once    sync.Once
	manager *Manager
)

// Singleton returns the singleton GC manager, initialized with the given store.
// The first call initializes it; subsequent calls return the cached instance.
func Singleton(s cvev2store.Store) *Manager {
	once.Do(func() {
		manager = New(s)
	})
	return manager
}
