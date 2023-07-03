package datastore

import (
	pgStore "github.com/stackrox/rox/central/events/datastore/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

// Singleton returns a datastore instance to handle events.
func Singleton() DataStore {
	once.Do(func() {
		ds = New(pgStore.New(globaldb.GetPostgres()))
	})
	return ds
}
