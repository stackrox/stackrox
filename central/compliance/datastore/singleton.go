package datastore

import (
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
		dbStore := pgStore.NewStore(globaldb.GetPostgres())
		dsInstance = NewDataStore(dbStore, NewSacFilter())
	})
	return dsInstance
}
