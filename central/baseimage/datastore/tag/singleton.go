package tag

import (
	tagStore "github.com/stackrox/rox/central/baseimage/store/tag/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	ds   DataStore
)

// Singleton returns the global datastore instance for base image tags.
func Singleton() DataStore {
	once.Do(func() {
		ds = New(tagStore.New(globaldb.GetPostgres()))
	})
	return ds
}
