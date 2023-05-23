package datastore

import (
	"github.com/stackrox/rox/central/compliance/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/compliance/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once       sync.Once
	dsInstance DataStore
)

// Singleton returns the compliance DataStore singleton.
func Singleton() DataStore {
	once.Do(func() {
		var dbStore store.Store
		dbStore = pgStore.NewStore(globaldb.GetPostgres())
		dsInstance = NewDataStore(dbStore, NewSacFilter())
	})
	return dsInstance
}
