package registry

import "github.com/stackrox/stackrox/pkg/sync"

var (
	once   sync.Once
	rStore *Store
)

// Singleton returns a singleton of the registry storage.
func Singleton() *Store {
	once.Do(func() {
		rStore = NewRegistryStore(nil)
	})
	return rStore
}
