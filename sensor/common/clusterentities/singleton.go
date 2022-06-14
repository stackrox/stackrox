package clusterentities

import "github.com/stackrox/rox/pkg/sync"

var (
	storeInstance     *Store
	storeInstanceInit sync.Once
)

// StoreInstance returns the singleton instance for the cluster entity store.
func StoreInstance() *Store {
	storeInstanceInit.Do(func() {
		storeInstance = NewStore()
	})
	return storeInstance
}
