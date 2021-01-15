package store

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	store Store
	once  sync.Once
)

// Singleton returns the singleton data store for cluster init bundles.
func Singleton() Store {
	once.Do(func() {
		store = NewInMemory()
	})
	return store
}
