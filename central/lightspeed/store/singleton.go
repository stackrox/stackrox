package store

import "github.com/stackrox/rox/pkg/sync"

var (
	instance     *Store
	instanceInit sync.Once
)

// Singleton returns the singleton instance of the Lightspeed store.
func Singleton() *Store {
	instanceInit.Do(func() {
		instance = New()
	})
	return instance
}
