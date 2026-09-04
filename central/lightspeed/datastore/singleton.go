package datastore

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance DataStore
)

// Singleton returns the singleton instance of the lightspeed DataStore.
func Singleton() DataStore {
	once.Do(func() {
		instance = New()
	})
	return instance
}
