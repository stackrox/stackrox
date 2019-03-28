package store

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	instance     Store
	instanceInit sync.Once
)

// Singleton returns the singleton instance of the license key store.
func Singleton() Store {
	instanceInit.Do(func() {
		store, err := New(globaldb.GetGlobalDB())
		utils.Must(err)
		instance = store
	})
	return instance
}
