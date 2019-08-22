package store

import (
	"time"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	singleton     Store
	singletonInit sync.Once
)

// Singleton returns a singleton of the Store class
func Singleton() Store {
	singletonInit.Do(func() {
		cache := expiringcache.NewExpiringCache(time.Hour)
		store, err := New(globaldb.GetGlobalDB(), cache)
		utils.Must(err)
		singleton = store
	})
	return singleton
}
