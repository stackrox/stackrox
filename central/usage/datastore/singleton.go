package datastore

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ds   DataStore
	once sync.Once

	log = logging.LoggerForModule()
)

// Singleton returns the singleton providing access to the usage store.
func Singleton() DataStore {
	once.Do(func() {
		ds = New(nil)
	})
	return ds
}
