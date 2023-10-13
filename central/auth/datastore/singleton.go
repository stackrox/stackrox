package datastore

import (
	pgStore "github.com/stackrox/rox/central/auth/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

// Singleton provides a singleton auth machine to machine DataStore.
func Singleton() DataStore {
	once.Do(func() {
		ds = New(pgStore.New(globaldb.GetPostgres()))
	})
	return ds
}
