package idmap

import "github.com/stackrox/stackrox/pkg/sync"

var (
	storageInstanceInit sync.Once
	storageInstance     Storage
)

// StorageSingleton retrieves the global ID map storage.
func StorageSingleton() Storage {
	storageInstanceInit.Do(func() {
		storageInstance = newSharedIDMapStorage()
	})
	return storageInstance
}
