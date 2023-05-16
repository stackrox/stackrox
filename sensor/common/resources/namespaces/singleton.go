package namespaces

import "github.com/stackrox/rox/pkg/sync"

var (
	once sync.Once

	nsStore *NamespaceStore
)

func initialize() {
	nsStore = newNamespaceStore()
}

// Singleton provides the interface for getting annotation values with a memory backed implementation.
func Singleton() *NamespaceStore {
	once.Do(initialize)
	return nsStore
}
