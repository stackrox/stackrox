package gc

import (
	"sync"

	cvev2datastore "github.com/stackrox/rox/central/cve/image/v2/datastore"
)

var (
	once    sync.Once
	manager *Manager
)

// Singleton returns the singleton GC manager, initialized with the given datastore.
// The first call initializes it; subsequent calls return the cached instance.
func Singleton(ds cvev2datastore.DataStore) *Manager {
	once.Do(func() {
		manager = New(ds)
	})
	return manager
}
